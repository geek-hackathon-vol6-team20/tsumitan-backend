package auth

import (
	"context"
	"crypto/rsa"
	"crypto/x509"

	// "encoding/base64" // Not directly used if jwt library handles header decoding
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

const UserIDContextKey = "userID"

var (
	expectedIssuer   = "https://securetoken.google.com/YOUR_PROJECT_ID" // TODO: Configure for your Firebase project
	expectedAudience = "YOUR_PROJECT_ID"                                // TODO: Configure for your Firebase project
)

const googlePublicKeysURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

type CustomClaims struct {
	jwt.RegisteredClaims
	// Add any other custom claims you expect from Firebase if necessary
	// Standard claims like 'email', 'email_verified', 'firebase' (with 'identities', 'sign_in_provider')
	// are often accessed via claims.Extra if not defined here.
}

var (
	googlePublicKeys      map[string]*rsa.PublicKey
	googlePublicKeysMutex = &sync.Mutex{} // Simpler mutex for basic protection
	lastKeyFetchTime      time.Time
	keyCacheDuration      = 24 * time.Hour // Fetch keys at most once per 24 hours
)

// getGooglePublicKey fetches and caches Google's public RSA key based on the token's Key ID (kid).
func getGooglePublicKey(kid string) (*rsa.PublicKey, error) {
	googlePublicKeysMutex.Lock()
	defer googlePublicKeysMutex.Unlock()

	if googlePublicKeys == nil || time.Since(lastKeyFetchTime) > keyCacheDuration {
		// Fetch and update keys
		// fmt.Println("Fetching Google public keys...")
		resp, err := http.Get(googlePublicKeysURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch Google public keys: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch Google public keys: status %s", resp.Status)
		}

		var rawKeys map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&rawKeys); err != nil {
			return nil, fmt.Errorf("failed to decode Google public keys: %w", err)
		}

		parsedKeys := make(map[string]*rsa.PublicKey)
		for keyID, certStr := range rawKeys {
			block, _ := pem.Decode([]byte(certStr))
			if block == nil {
				// Log or handle error for specific key, but try to parse others
				fmt.Printf("Warning: failed to parse PEM block for kid: %s\n", keyID)
				continue
			}
			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				fmt.Printf("Warning: failed to parse certificate for kid %s: %v\n", keyID, err)
				continue
			}
			rsaPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
			if !ok {
				fmt.Printf("Warning: public key for kid %s is not RSA type\n", keyID)
				continue
			}
			parsedKeys[keyID] = rsaPubKey
		}
		if len(parsedKeys) == 0 && len(rawKeys) > 0 {
			return nil, errors.New("no valid RSA public keys found after parsing all fetched keys")
		}
		if len(parsedKeys) == 0 {
			return nil, errors.New("no public keys were successfully parsed")
		}

		googlePublicKeys = parsedKeys
		lastKeyFetchTime = time.Now()
		// fmt.Println("Successfully fetched and cached Google public keys.")
	}

	if key, ok := googlePublicKeys[kid]; ok {
		return key, nil
	}
	// If key not found, it might be a new key, try refreshing cache once more immediately.
	// This is a simple strategy; more robust would be to check Cache-Control/Expires headers from Google.
	// For this simplified version, we don't re-fetch immediately if kid is not found after a recent fetch.
	return nil, fmt.Errorf("public key for kid '%s' not found in cache", kid)
}

// verifyTokenFirebase implements Firebase-specific JWT validation.
func verifyTokenFirebase(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("kid header not found in token")
		}

		if alg, ok := token.Header["alg"].(string); !ok || alg != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v. Expected RS256", token.Header["alg"])
		}

		return getGooglePublicKey(kid)
	})

	if err != nil {
		return "", fmt.Errorf("token parsing or validation error: %w", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.Issuer != expectedIssuer {
			return "", fmt.Errorf("invalid issuer. expected '%s', got '%s'", expectedIssuer, claims.Issuer)
		}
		if !claims.IsForAudience(expectedAudience) {
			return "", fmt.Errorf("invalid audience. expected '%s', got '%s'", expectedAudience, claims.Audience)
		}
		if claims.Subject == "" {
			return "", errors.New("subject (user ID) claim is missing or empty")
		}
		// fmt.Printf("Token validated successfully for user: %s (sub)\n", claims.Subject)
		return claims.Subject, nil
	}

	return "", errors.New("invalid token or claims type")
}

// AuthMiddleware is an Echo middleware for handling authentication.
func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get("Authorization")
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Missing Authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid Authorization header format. Expected 'Bearer <token>'."})
		}
		tokenString := parts[1]

		userID, err := verifyTokenFirebase(tokenString)
		if err != nil {
			c.Logger().Errorf("Error verifying Firebase token: %v", err)
			// Provide a generic error message to the client
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "Invalid or expired token"})
		}

		if userID == "" {
			// This case should ideally be caught by errors from verifyTokenFirebase, but as a safeguard:
			c.Logger().Error("UserID is empty after successful token verification, this should not happen.")
			return c.JSON(http.StatusInternalServerError, map[string]string{"message": "Unable to identify user from token after verification"})
		}

		ctxWithUserID := context.WithValue(c.Request().Context(), UserIDContextKey, userID)
		reqWithUserID := c.Request().WithContext(ctxWithUserID)
		c.SetRequest(reqWithUserID)
		c.Set(UserIDContextKey, userID) // Also set directly for easier access

		return next(c)
	}
}
