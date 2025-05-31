package database

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/vladimiradmaev/diabetes-helper/internal/config"
	"github.com/vladimiradmaev/diabetes-helper/internal/database/migrations"
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
	BreadUnits   float64
	Confidence   float64
	AnalysisText string
	UsedProvider string // "gemini" or "openai"
	InsulinRatio float64
	InsulinUnits float64
}

type FoodAnalysisCorrection struct {
	gorm.Model
	UserID          uint
	User            User
	OriginalCarbs   float64
	CorrectedCarbs  float64
	BreadUnits      float64
	OriginalWeight  float64
	CorrectedWeight float64
	ImageURL        string
	AnalysisText    string
	UsedProvider    string // "gemini" or "openai"
	Confidence      float64
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

	// Get the directory of the current file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return nil, fmt.Errorf("failed to get current file path")
	}
	migrationsDir := filepath.Join(filepath.Dir(filename), "migrations")

	// Load and run migrations
	if err := migrations.LoadSQLMigrations(db, migrationsDir); err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	if err := migrations.RunMigrations(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Auto-migrate the schema for models that don't have explicit migrations
	if err := db.AutoMigrate(&User{}, &FoodAnalysis{}, &FoodAnalysisCorrection{}, &BloodSugarRecord{}, &InsulinRatio{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	log.Println("Database connection established and migrations completed")
	return db, nil
}
