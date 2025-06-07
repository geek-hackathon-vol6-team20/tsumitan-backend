package server

import (
	"net/http"

	"tsumitan/internal/auth"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"https://*", "http://*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Register the AuthMiddleware
	// This will apply to all routes defined after this line.
	e.Use(auth.AuthMiddleware)

	e.GET("/", s.HelloWorldHandler)
	e.GET("/health", s.healthHandler)

	// /api以下をAPIのルートとして登録
	api := e.Group("/api")
	{
		api.POST("/search", s.SearchHandler)
		// api.GET("/review/pending", s.GetPendingReviewsHandler)
		// api.PATCH("/review", s.ReviewHandler)
		// api.GET("/review/history", s.ReviewHistoryHandler)
		// api.GET("/word/:word", s.GetWordHandler)
	}

	return e
}
