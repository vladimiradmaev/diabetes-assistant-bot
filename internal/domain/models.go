package domain

import (
	"time"
)

// User represents a telegram user in the system
type User struct {
	ID                uint
	CreatedAt         time.Time
	UpdatedAt         time.Time
	DeletedAt         *time.Time
	TelegramID        int64
	Username          string
	FirstName         string
	LastName          string
	ActiveInsulinTime int // Time in minutes
}

// FoodAnalysis represents a food analysis result
type FoodAnalysis struct {
	ID           uint
	UserID       uint
	ImageURL     string
	Weight       float64
	Carbs        float64
	BreadUnits   float64
	Confidence   float64
	AnalysisText string
	UsedProvider string
	InsulinUnits float64
	InsulinRatio float64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// BloodSugarRecord represents a blood sugar measurement
type BloodSugarRecord struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	UserID    uint
	User      User
	Value     float64
	Timestamp time.Time
}

// InsulinRatio represents insulin ratio for a specific time period
type InsulinRatio struct {
	ID        uint
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
	UserID    uint
	User      User
	StartTime string  // Format: "HH:MM"
	EndTime   string  // Format: "HH:MM"
	Ratio     float64 // Insulin units per XE
}
