package server

import (
	"io"
	"log"
	"net/http"
	"tsumitan/internal/auth"

	"github.com/labstack/echo/v4"
)

// SearchRequest represents the request body for search endpoint
type SearchRequest struct {
	Word string `json:"word"`
}

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Message string `json:"message"`
}

func (s *Server) HelloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, s.db.Health())
}

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

	// Search word meaning from external API
	client := &http.Client{}
	url := "https://api.excelapi.org/dictionary/enja?word=" + word

	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Failed to fetch word meaning: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "辞書APIエラー",
		})
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("Warning: failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read API response: %v", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{
			Message: "辞書APIレスポンス読み込みエラー",
		})
	}

	meaning := string(body)

	//単語の意味がない時に404 Not Foundを返す
	if meaning == "" {
		log.Printf("No meaning found for word: %s", word)
		return c.JSON(http.StatusNotFound, ErrorResponse{
			Message: "単語の意味が見つかりません",
		})
	}

	log.Printf("Word meaning fetched for user %s, word: %s (no search count increment)", userID, word)

	// Return word meaning without incrementing search count
	return c.JSON(http.StatusOK, map[string]any{
		"word":     word,
		"meanings": meaning,
	})
}

type PendingResponse struct {
	Word        string `json:"word"`
	SearchCount int    `json:"search_count"`
}

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
