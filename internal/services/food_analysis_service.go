package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"gorm.io/gorm"
)

type FoodAnalysisService struct {
	aiService *AIService
	db        *gorm.DB
}

const (
	highConfidenceThreshold   = 0.8
	mediumConfidenceThreshold = 0.6
	lowConfidenceThreshold    = 0.4
)

func NewFoodAnalysisService(aiService *AIService, db *gorm.DB) *FoodAnalysisService {
	return &FoodAnalysisService{
		aiService: aiService,
		db:        db,
	}
}

func (s *FoodAnalysisService) AnalyzeFood(ctx context.Context, userID uint, imageURL string, weight float64, useOpenAI bool) (*database.FoodAnalysis, error) {
	result, err := s.aiService.AnalyzeFoodImage(ctx, imageURL, weight, useOpenAI)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze food image: %w", err)
	}

	// Use the weight from the AI result if no weight was provided
	if weight <= 0 && result.Weight > 0 {
		weight = result.Weight
	}

	// Convert confidence string to float64
	var confidence float64
	switch strings.ToLower(result.Confidence) {
	case "high":
		confidence = 0.9
	case "medium":
		confidence = 0.6
	case "low":
		confidence = 0.3
	default:
		confidence = 0.5
	}

	// Calculate bread units (ХЕ) - 1 ХЕ = 12g of carbs
	breadUnits := result.Carbs / 12.0

	// Get current time to find the appropriate insulin ratio
	now := time.Now()
	currentTime := fmt.Sprintf("%02d:%02d", now.Hour(), now.Minute())

	// Get user's insulin ratios
	var ratios []database.InsulinRatio
	if err := s.db.Where("user_id = ?", userID).Find(&ratios).Error; err != nil {
		return nil, fmt.Errorf("failed to get insulin ratios: %w", err)
	}

	// Find the appropriate ratio for current time
	var insulinRatio float64
	for _, r := range ratios {
		if currentTime >= r.StartTime && currentTime <= r.EndTime {
			insulinRatio = r.Ratio
			break
		}
	}

	// Calculate insulin units (ХЕ * ratio)
	insulinUnits := breadUnits * insulinRatio

	analysis := &database.FoodAnalysis{
		UserID:       userID,
		ImageURL:     imageURL,
		Weight:       weight,
		Carbs:        result.Carbs,
		BreadUnits:   breadUnits,
		Confidence:   confidence,
		AnalysisText: result.AnalysisText,
		UsedProvider: "openai",
		InsulinRatio: insulinRatio,
		InsulinUnits: insulinUnits,
	}
	if !useOpenAI {
		analysis.UsedProvider = "gemini"
	}

	if err := s.db.WithContext(ctx).Create(analysis).Error; err != nil {
		return nil, fmt.Errorf("failed to save analysis: %w", err)
	}

	return analysis, nil
}

func (s *FoodAnalysisService) GetUserAnalyses(ctx context.Context, userID uint) ([]database.FoodAnalysis, error) {
	var analyses []database.FoodAnalysis
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&analyses).Error; err != nil {
		return nil, fmt.Errorf("failed to get user analyses: %w", err)
	}
	return analyses, nil
}

func (s *FoodAnalysisService) SaveCorrection(ctx context.Context, userID uint, originalAnalysis *database.FoodAnalysis, correctedCarbs, correctedWeight float64) error {
	correction := &database.FoodAnalysisCorrection{
		UserID:          userID,
		OriginalCarbs:   originalAnalysis.Carbs,
		CorrectedCarbs:  correctedCarbs,
		OriginalWeight:  originalAnalysis.Weight,
		CorrectedWeight: correctedWeight,
		ImageURL:        originalAnalysis.ImageURL,
		AnalysisText:    originalAnalysis.AnalysisText,
		UsedProvider:    originalAnalysis.UsedProvider,
		Confidence:      originalAnalysis.Confidence,
	}
	if err := s.db.Create(correction).Error; err != nil {
		return fmt.Errorf("failed to save correction: %w", err)
	}
	return nil
}

func (s *FoodAnalysisService) GetUserCorrections(ctx context.Context, userID uint) ([]*database.FoodAnalysisCorrection, error) {
	var corrections []*database.FoodAnalysisCorrection
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Find(&corrections).Error; err != nil {
		return nil, fmt.Errorf("failed to get corrections: %w", err)
	}
	return corrections, nil
}
