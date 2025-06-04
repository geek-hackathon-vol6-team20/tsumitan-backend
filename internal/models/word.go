package models

import (
	"time"
)

type Word struct {
	UserID       string    `gorm:"primaryKey"`
	Word         string    `gorm:"primaryKey"`
	SearchCount  int
	ReviewCount  int
	LastReviewed time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
