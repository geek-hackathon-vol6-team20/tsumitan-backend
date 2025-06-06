package models

import (
	"time"
)

type Word struct {
	UserID       string    `gorm:"primaryKey" json:"user_id"`
	Word         string    `gorm:"primaryKey" json:"word"`
	SearchCount  int       `json:"search_count"`
	ReviewCount  int       `json:"review_count"`
	LastReviewed time.Time `json:"last_reviewed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
