package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

const googlePublicKeysURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// 公開鍵のキャッシュ構造
type publicKeyCache struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
	mutex     sync.RWMutex
}

var keyCache = &publicKeyCache{
	keys: make(map[string]*rsa.PublicKey),
}

type FirebaseClaims struct {
	jwt.RegisteredClaims
	AuthTime int64 `json:"auth_time"`
}

func getPublicKey(kid string) (*rsa.PublicKey, error) {
	keyCache.mutex.RLock()

	// キャッシュが有効で、キーが存在する場合はキャッシュから返す
	if time.Now().Before(keyCache.expiresAt) {
		if key, exists := keyCache.keys[kid]; exists {
			keyCache.mutex.RUnlock()
			return key, nil
		}
	}
	keyCache.mutex.RUnlock()

	// キャッシュが無効またはキーが存在しない場合は取得
	return fetchAndCachePublicKeys(kid)
}

func fetchAndCachePublicKeys(kid string) (*rsa.PublicKey, error) {
	resp, err := http.Get(googlePublicKeysURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch public keys: %w", err)
	}
	defer resp.Body.Close()

	// Cache-Controlヘッダーからmax-age値を取得
	cacheControl := resp.Header.Get("Cache-Control")
	maxAge := int64(3600) // デフォルト1時間
	if cacheControl != "" {
		parts := strings.Split(cacheControl, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "max-age=") {
				if age, err := strconv.ParseInt(part[8:], 10, 64); err == nil {
					maxAge = age
				}
			}
		}
	}

	var rawKeys map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&rawKeys); err != nil {
		return nil, fmt.Errorf("failed to decode public keys: %w", err)
	}

	keyCache.mutex.Lock()
	defer keyCache.mutex.Unlock()

	// 新しいキーマップを作成
	newKeys := make(map[string]*rsa.PublicKey)
	for keyID, certStr := range rawKeys {
		key, err := parsePublicKey(certStr)
		if err != nil {
			log.Printf("Warning: failed to parse public key for kid '%s': %v", keyID, err)
			continue
		}
		newKeys[keyID] = key
	}

	// キャッシュを更新
	keyCache.keys = newKeys
	keyCache.expiresAt = time.Now().Add(time.Duration(maxAge) * time.Second)

	// 要求されたキーを返す
	if key, exists := newKeys[kid]; exists {
		return key, nil
	}

	return nil, fmt.Errorf("public key for kid '%s' not found", kid)
}

func parsePublicKey(certStr string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(certStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not RSA type")
	}

	return rsaPubKey, nil
}

func verifyFirebaseToken(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &FirebaseClaims{}, func(token *jwt.Token) (any, error) {
		// alg（アルゴリズム）の検証 - "RS256" である必要がある
		if alg, ok := token.Header["alg"].(string); !ok || alg != "RS256" {
			return nil, fmt.Errorf("unexpected signing method: expected RS256, got %v", token.Header["alg"])
		}

		// kid（キー ID）の検証
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		return getPublicKey(kid)
	})

	if err != nil {
		return "", fmt.Errorf("token parsing failed: %w", err)
	}

	claims, ok := token.Claims.(*FirebaseClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	if err := validateClaims(claims); err != nil {
		return "", fmt.Errorf("claims validation failed: %w", err)
	}

	return claims.Subject, nil
}

func validateClaims(claims *FirebaseClaims) error {
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if projectID == "" {
		log.Fatal("FIREBASE_PROJECT_ID environment variable is not set")
	}

	now := time.Now().Unix()

	// exp（有効期限）の検証 - 将来の時刻である必要がある
	if claims.ExpiresAt == nil || claims.ExpiresAt.Unix() <= now {
		return fmt.Errorf("token has expired")
	}

	// iat（発行時刻）の検証 - 過去の時刻である必要がある
	if claims.IssuedAt == nil || claims.IssuedAt.Unix() > now {
		return fmt.Errorf("token used before issued")
	}

	// iatとexpの間隔が合理的であることを確認（最大24時間）
	if claims.ExpiresAt.Unix()-claims.IssuedAt.Unix() > 24*3600 {
		return fmt.Errorf("token validity period too long")
	}

	// iss（発行元）の検証
	expectedIssuer := fmt.Sprintf("https://securetoken.google.com/%s", projectID)
	if claims.Issuer != expectedIssuer {
		return fmt.Errorf("invalid issuer: expected %s, got %s", expectedIssuer, claims.Issuer)
	}

	// aud（対象）の検証 - Firebase プロジェクトの ID である必要がある
	if len(claims.Audience) == 0 || claims.Audience[0] != projectID {
		return fmt.Errorf("invalid audience: expected %s", projectID)
	}

	// sub（件名）の検証 - 空でない文字列である必要がある
	if claims.Subject == "" {
		return fmt.Errorf("missing or empty subject")
	}

	// auth_time（認証時間）の検証 - 過去の時刻である必要がある
	if claims.AuthTime == 0 || claims.AuthTime > now {
		return fmt.Errorf("invalid auth_time: must be in the past")
	}

	// auth_timeがiatより古いか同じであることを確認
	if claims.AuthTime > claims.IssuedAt.Unix() {
		return fmt.Errorf("auth_time cannot be after issued time")
	}

	// auth_timeが合理的な範囲内（過去30日以内）であることを確認
	if claims.AuthTime < now-30*24*3600 {
		return fmt.Errorf("auth_time too old")
	}

	return nil
}

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// APP_ENV環境変数をチェック
		appEnv := os.Getenv("APP_ENV")
		if appEnv == "" {
			log.Printf("Warning: APP_ENV environment variable is not set, authentication will be enforced")
		} else if appEnv == "local" {
			log.Printf("Local environment detected: bypassing authentication for request from %s", c.RealIP())
			// ローカル環境では認証をバイパスし、ダミーのユーザーIDを設定
			dummyUserID := "local-user"
			ctx := context.WithValue(c.Request().Context(), UserIDContextKey, dummyUserID)
			c.SetRequest(c.Request().WithContext(ctx))
			c.Set(string(UserIDContextKey), dummyUserID)
			return next(c)
		}

		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			log.Printf("Authentication failed: missing authorization header from %s", c.RealIP())
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			log.Printf("Authentication failed: invalid authorization header format from %s", c.RealIP())
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization header"})
		}

		userID, err := verifyFirebaseToken(parts[1])
		if err != nil {
			log.Printf("Authentication failed: token verification error from %s: %v", c.RealIP(), err)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		}

		log.Printf("Authentication successful for user %s from %s", userID, c.RealIP())

		ctx := context.WithValue(c.Request().Context(), UserIDContextKey, userID)
		c.SetRequest(c.Request().WithContext(ctx))
		c.Set(string(UserIDContextKey), userID)

		return next(c)
	}
}
