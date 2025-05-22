package services

import (
	"context"
	"fmt"

	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"gorm.io/gorm"
)

type FoodAnalysisService struct {
	aiService *AIService
	db        *gorm.DB
}

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

	analysis := &database.FoodAnalysis{
		UserID:       userID,
		ImageURL:     imageURL,
		Weight:       weight,
		Carbs:        result.Carbs,
		Confidence:   result.Confidence,
		AnalysisText: result.AnalysisText,
		UsedProvider: "openai",
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
