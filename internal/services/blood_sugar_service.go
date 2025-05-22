package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"gorm.io/gorm"
)

type BloodSugarService struct {
	db *gorm.DB
}

func NewBloodSugarService(db *gorm.DB) *BloodSugarService {
	return &BloodSugarService{
		db: db,
	}
}

func (s *BloodSugarService) AddRecord(ctx context.Context, userID uint, value float64) error {
	record := &database.BloodSugarRecord{
		UserID:    userID,
		Value:     value,
		Timestamp: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(record).Error; err != nil {
		return fmt.Errorf("failed to create blood sugar record: %w", err)
	}

	return nil
}

func (s *BloodSugarService) GetUserRecords(ctx context.Context, userID uint) ([]database.BloodSugarRecord, error) {
	var records []database.BloodSugarRecord
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("timestamp DESC").
		Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to get user blood sugar records: %w", err)
	}
	return records, nil
}
