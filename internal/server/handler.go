package server

import (
	"log"
	"net/http"
	"tsumitan/internal/auth"

	"github.com/labstack/echo/v4"
)

// SearchRequest represents the request body for search endpoint
type SearchRequest struct {
	Word string `json:"word" validate:"required"`
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

	// Return success response
	return c.JSON(http.StatusOK, map[string]interface{}{})
}
