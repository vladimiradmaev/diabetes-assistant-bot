package database

import (
	"fmt"
	"log"
	"time"

	"github.com/vladimiradmaev/diabetes-helper/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	TelegramID        int64 `gorm:"uniqueIndex"`
	Username          string
	FirstName         string
	LastName          string
	ActiveInsulinTime int `gorm:"default:0"` // Time in minutes
}

type FoodAnalysis struct {
	gorm.Model
	UserID       uint
	User         User
	ImageURL     string
	Weight       float64
	Carbs        float64
	Confidence   string
	AnalysisText string
	UsedProvider string // "gemini" or "openai"
}

type BloodSugarRecord struct {
	gorm.Model
	UserID    uint
	User      User
	Value     float64
	Timestamp time.Time
}

type InsulinRatio struct {
	gorm.Model
	UserID    uint
	User      User
	StartTime string  // Format: "HH:MM"
	EndTime   string  // Format: "HH:MM"
	Ratio     float64 // Insulin units per XE
}

func NewPostgresDB(cfg config.DBConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&User{}, &FoodAnalysis{}, &BloodSugarRecord{}, &InsulinRatio{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connection established and migrations completed")
	return db, nil
}
