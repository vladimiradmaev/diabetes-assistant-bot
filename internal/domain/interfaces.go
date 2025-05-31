package domain

import (
	"context"
	"time"
)

// UserService handles user-related operations
type UserService interface {
	GetOrCreateUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*User, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (*User, error)
}

// FoodAnalysisService handles food analysis operations
type FoodAnalysisService interface {
	SaveAnalysis(ctx context.Context, analysis *FoodAnalysis) error
	GetUserAnalyses(ctx context.Context, userID uint) ([]FoodAnalysis, error)
}

// BloodSugarService handles blood sugar record operations
type BloodSugarService interface {
	SaveRecord(ctx context.Context, record *BloodSugarRecord) error
	GetUserRecords(ctx context.Context, userID uint, start, end time.Time) ([]BloodSugarRecord, error)
}

// InsulinService handles insulin-related operations
type InsulinService interface {
	SaveRatio(ctx context.Context, ratio *InsulinRatio) error
	GetUserRatios(ctx context.Context, userID uint) ([]InsulinRatio, error)
	DeleteRatio(ctx context.Context, id uint) error
	GetActiveInsulinTime(ctx context.Context, userID uint) (int, error)
	SetActiveInsulinTime(ctx context.Context, userID uint, minutes int) error
}

// BotService handles telegram bot operations
type BotService interface {
	Start(ctx context.Context) error
	Stop()
}
