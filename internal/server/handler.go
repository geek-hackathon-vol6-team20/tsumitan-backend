package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"tsumitan/internal/auth"

	"github.com/labstack/echo/v4"
)

func (s *Server) HelloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, s.db.Health())
}

// キャッシュ生成用
var (
	wordCache = make(map[string]string)
	cacheMu   sync.RWMutex
)

type DictionaryResponse struct {
	Word     string `json:"word"`
	Meanings string `json:"meanings"`
}

// 単語の意味を取得する（キャッシュを用いる）
func FetchWordMeaning(word string) (string, error) {
	if word == "" {
		return "", fmt.Errorf("単語が指定されていません")
	}

	cacheMu.RLock()
	if meanings, found := wordCache[word]; found {
		cacheMu.RUnlock()
		return meanings, nil
	}
	cacheMu.RUnlock()

	// 辞書APIにリクエストを送信
	url := "https://api.excelapi.org/dictionary/enja?word=" + url.QueryEscape(word)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("辞書APIリクエスト失敗: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("response body close error: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("辞書APIステータスエラー: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("辞書APIレスポンス読み取りエラー: %w", err)
	}

	meanings := string(body)

	// キャッシュに保存
	cacheMu.Lock()
	wordCache[word] = meanings
	cacheMu.Unlock()

	return meanings, nil
}

// SearchRequest represents the request body for search endpoint
type SearchRequest struct {
	Word string `json:"word"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Message string `json:"message"`
}

// SearchHandler handles POST /api/search - records a word search and returns no meaning
func (s *Server) SearchHandler(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get(string(auth.UserIDContextKey)).(string)
	if !ok {
		log.Println("User ID not found in context")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "ユーザーIDが見つかりません",
		})
	}

	// Parse request body
	var req SearchRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind request: %v", err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "リクエスト不備",
		})
	}

	// Validate required fields
	if req.Word == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "必須フィールドが不足しています",
		})
	}

	// 単語の意味が存在するか確認
	meanings, err := FetchWordMeaning(req.Word)
	if err != nil || meanings == "" {
		log.Printf("意味取得失敗: %v", err)
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "意味の取得に失敗しました",
		})
	}

	// Record search in database
	if err := s.db.CreateOrUpdateWordSearch(userID, req.Word); err != nil {
		log.Printf("Failed to record search: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "サーバーエラー",
		})
	}

	log.Printf("Search recorded for user %s, word: %s", userID, req.Word)

	// Return success response (no meaning returned)
	return c.JSON(http.StatusOK, map[string]string{
		"message": "検索が記録されました",
	})
}

// GetWordMeaningHandler handles GET /api/search?word={word} - returns word meaning without incrementing search count
func (s *Server) GetWordMeaningHandler(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get(string(auth.UserIDContextKey)).(string)
	if !ok {
		log.Println("User ID not found in context")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "ユーザーIDが見つかりません",
		})
	}

	// Get word from query parameter
	word := c.QueryParam("word")
	if word == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "単語パラメータが必要です",
		})
	}

	meanings, err := FetchWordMeaning(word)
	if err != nil || meanings == "" {
		log.Printf("意味取得失敗: %v", err)
		return c.JSON(http.StatusNotFound, ErrorResponse{Message: "意味の取得に失敗しました"})
	}

	log.Printf("Word meaning fetched for user %s, word: %s (no search count increment)", userID, word)

	// Return word meaning without incrementing search count
	return c.JSON(http.StatusOK, DictionaryResponse{
		Word:     word,
		Meanings: meanings, // Assuming we return the first meaning
	})
}

type PendingResponse struct {
	Word        string `json:"word"`
	SearchCount int    `json:"search_count"`
}

// GetPendingReviewsHandler handles GET /api/review/pending - returns words pending review for the user
func (s *Server) GetPendingReviewsHandler(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get(string(auth.UserIDContextKey)).(string)
	if !ok {
		log.Println("User ID not found in context")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "ユーザーIDが見つかりません",
		})
	}

	// Fetch pending reviews from database
	// データベースから未レビューの単語を取得
	pendingReviews, err := s.db.PendingWordSearch(userID)
	if err != nil {
		log.Printf("Failed to fetch pending reviews: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "サーバーエラー",
		})
	}

	// Map database results to PendingResponse
	response := []PendingResponse{}

	for _, review := range pendingReviews {
		response = append(response, PendingResponse{
			Word:        review.Word,
			SearchCount: review.SearchCount,
		})
	}

	// Return filtered response
	return c.JSON(http.StatusOK, response)
}

type ReviewRequest struct {
	Word string `json:"word"`
}

// ReviewHandler handles PATCH /api/review - records a review for a word
func (s *Server) ReviewHandler(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get(string(auth.UserIDContextKey)).(string)
	if !ok {
		log.Println("User ID not found in context")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "ユーザーIDが見つかりません",
		})
	}

	// Parse request body
	var req ReviewRequest
	if err := c.Bind(&req); err != nil {
		log.Printf("Failed to bind request: %v", err)
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "リクエスト不備",
		})
	}

	// Validate required fields
	if req.Word == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "必須フィールドが不足しています",
		})
	}

	// Update review count in database
	if err := s.db.UpdateWordReview(userID, req.Word); err != nil {
		log.Printf("Failed to update review: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "サーバーエラー",
		})
	}

	log.Printf("Review updated for user %s, word: %s", userID, req.Word)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "復習が記録されました。"})
}

type ReviewHistoryResponse struct {
	Word         string `json:"word"`
	SearchCount  int    `json:"search_count"`
	ReviewCount  int    `json:"review_count"`
	LastReviewed string `json:"last_reviewed"`
}

// ReviewHistoryHandler handles GET /api/review/history - returns review history for the user
func (s *Server) ReviewHistoryHandler(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get(string(auth.UserIDContextKey)).(string)
	if !ok {
		log.Println("User ID not found in context")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "ユーザーIDが見つかりません",
		})
	}

	// Fetch pending reviews from database
	// データベースから未レビューの単語を取得
	reviewedRecords, err := s.db.ReviewedWordSearch(userID)
	if err != nil {
		log.Printf("Failed to fetch pending reviews: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "サーバーエラー",
		})
	}

	// Map database results to PendingResponse
	response := []ReviewHistoryResponse{}

	for _, review := range reviewedRecords {
		response = append(response, ReviewHistoryResponse{
			Word:         review.Word,
			SearchCount:  review.SearchCount,
			ReviewCount:  review.ReviewCount,
			LastReviewed: review.LastReviewed.String(),
		})
	}

	// Return filtered response
	return c.JSON(http.StatusOK, response)
}

type WordDetailResponse struct {
	Word         string `json:"word"`
	SearchCount  int    `json:"search_count"`
	ReviewCount  int    `json:"review_count"`
	LastReviewed string `json:"last_reviewed"`
}

// GetWordHandler handles GET /api/word/:word - returns detailed word info for the user
func (s *Server) GetWordHandler(c echo.Context) error {
	// Get user ID from context (set by auth middleware)
	userID, ok := c.Get(string(auth.UserIDContextKey)).(string)
	if !ok {
		log.Println("User ID not found in context")
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "ユーザーIDが見つかりません",
		})
	}

	word := c.Param("word")
	if word == "" {
		return c.JSON(http.StatusBadRequest, ErrorResponse{
			Message: "単語が指定されていません",
		})
	}

	// Fetch word from database
	wordRecord, err := s.db.GetWordInfo(userID, word)
	if wordRecord == nil {
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "単語が見つかりません",
		})
	}
	if err != nil {
		log.Printf("Failed to fetch word: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "サーバーエラー",
		})
	}

	// Map database results to PendingResponse
	response := WordDetailResponse{
		Word:         wordRecord.Word,
		SearchCount:  wordRecord.SearchCount,
		ReviewCount:  wordRecord.ReviewCount,
		LastReviewed: wordRecord.LastReviewed.String(),
	}

	// Return filtered response
	return c.JSON(http.StatusOK, response)
}
