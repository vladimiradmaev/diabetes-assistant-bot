package interfaces

import (
	"context"

	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"github.com/vladimiradmaev/diabetes-helper/internal/services"
)

// UserServiceInterface defines the contract for user operations
type UserServiceInterface interface {
	RegisterUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*database.User, error)
	GetUserByTelegramID(ctx context.Context, telegramID int64) (*database.User, error)
}

// FoodAnalysisServiceInterface defines the contract for food analysis operations
type FoodAnalysisServiceInterface interface {
	AnalyzeFood(ctx context.Context, userID uint, imageURL string, weight float64) (*database.FoodAnalysis, error)
	GetUserAnalyses(ctx context.Context, userID uint) ([]database.FoodAnalysis, error)
}

// BloodSugarServiceInterface defines the contract for blood sugar operations
type BloodSugarServiceInterface interface {
	AddRecord(ctx context.Context, userID uint, value float64) error
	GetUserRecords(ctx context.Context, userID uint) ([]database.BloodSugarRecord, error)
}

// InsulinServiceInterface defines the contract for insulin operations
type InsulinServiceInterface interface {
	AddRatio(ctx context.Context, userID uint, startTime, endTime string, ratio float64) error
	GetUserRatios(ctx context.Context, userID uint) ([]database.InsulinRatio, error)
	DeleteRatio(ctx context.Context, userID uint, ratioID uint) error
	UpdateRatio(ctx context.Context, userID uint, ratioID uint, startTime, endTime string, ratio float64) error
	GetActiveInsulinTime(ctx context.Context, userID uint) (int, error)
	SetActiveInsulinTime(ctx context.Context, userID uint, minutes int) error
}

// AIServiceInterface defines the contract for AI operations
type AIServiceInterface interface {
	AnalyzeFoodImage(ctx context.Context, imageURL string, weight float64) (*services.FoodAnalysisResult, error)
}
