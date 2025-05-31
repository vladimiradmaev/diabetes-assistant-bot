package domain

import (
	"time"

	"gorm.io/gorm"
)

// User represents a telegram user in the system
type User struct {
	gorm.Model
	TelegramID        int64 `gorm:"uniqueIndex"`
	Username          string
	FirstName         string
	LastName          string
	ActiveInsulinTime int `gorm:"default:0"` // Time in minutes
}

// FoodAnalysis represents a food analysis result
type FoodAnalysis struct {
	ID           uint `gorm:"primaryKey"`
	UserID       uint `gorm:"index"`
	ImageURL     string
	Weight       float64
	Carbs        float64
	BreadUnits   float64 `gorm:"column:bread_units"`
	Confidence   float64
	AnalysisText string
	UsedProvider string
	InsulinUnits float64 `gorm:"column:insulin_units"`
	InsulinRatio float64 `gorm:"column:insulin_ratio"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// BloodSugarRecord represents a blood sugar measurement
type BloodSugarRecord struct {
	gorm.Model
	UserID    uint
	User      User
	Value     float64
	Timestamp time.Time
}

// InsulinRatio represents insulin ratio for a specific time period
type InsulinRatio struct {
	gorm.Model
	UserID    uint
	User      User
	StartTime string  // Format: "HH:MM"
	EndTime   string  // Format: "HH:MM"
	Ratio     float64 // Insulin units per XE
}
