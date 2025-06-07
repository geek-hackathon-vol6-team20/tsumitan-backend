package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"tsumitan/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Service represents a service that interacts with a database.
type Service interface {
	Health() map[string]string
	Close() error
	Migrate() error
	// Word operations
	CreateOrUpdateWordSearch(userID, word string) error
	PendingWordSearch(userID string) ([]models.Word, error)
}

type service struct {
	db *gorm.DB
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
)

func New() Service {
	if dbInstance != nil {
		return dbInstance
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s TimeZone=UTC",
		host, username, password, database, port, schema)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	sqlDB, err := s.db.DB()
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("failed to get underlying DB: %v", err)
		log.Printf("failed to get underlying DB: %v", err)
		return stats
	}

	err = sqlDB.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("db down: %v", err)
		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"

	dbStats := sqlDB.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	if dbStats.OpenConnections > 40 {
		stats["message"] = "The database is experiencing heavy load."
	}
	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}
	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
func (s *service) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		log.Printf("Error getting DB instance for closing: %v", err)
		return err
	}
	log.Printf("Disconnected from database: %s", database)
	return sqlDB.Close()
}

// Migrate performs database migration for the Word model.
func (s *service) Migrate() error {
	log.Println("Migrating database...")
	err := s.db.AutoMigrate(&models.Word{})
	if err != nil {
		log.Printf("Database migration failed: %v", err)
		return err
	}
	log.Println("Database migration completed.")
	return nil
}

// CreateOrUpdateWordSearch creates a new word record or increments search_count if it already exists
func (s *service) CreateOrUpdateWordSearch(userID, word string) error {
	var existingWord models.Word

	// Try to find existing record
	result := s.db.Where("user_id = ? AND word = ?", userID, word).First(&existingWord)

	if result.Error != nil {
		// Check if it's a "record not found" error using GORM's errors
		if result.Error == gorm.ErrRecordNotFound {
			// Create new record
			newWord := models.Word{
				UserID:      userID,
				Word:        word,
				SearchCount: 1,
				ReviewCount: 0,
			}
			return s.db.Create(&newWord).Error
		}
		// Other error occurred
		return result.Error
	}

	// Update existing record
	return s.db.Model(&existingWord).Update("search_count", existingWord.SearchCount+1).Error
}

func (s *service) PendingWordSearch(userID string) ([]models.Word, error) {
	var words []models.Word

	// Query to fetch records where ReviewCount = 0
	err := s.db.Where("user_id = ? AND review_count = ?", userID, 0).Find(&words).Error
	if err != nil {
		log.Printf("Error fetching pending words for user %s: %v", userID, err)
		return nil, err
	}

	return words, nil
}
