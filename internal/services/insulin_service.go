package services

import (
	"context"
	"fmt"
	"time"

	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"gorm.io/gorm"
)

type InsulinService struct {
	db *gorm.DB
}

func NewInsulinService(db *gorm.DB) *InsulinService {
	return &InsulinService{
		db: db,
	}
}

func (s *InsulinService) AddRatio(ctx context.Context, userID uint, startTime, endTime string, ratio float64) error {
	// Validate time format
	if _, err := time.Parse("15:04", startTime); err != nil {
		return fmt.Errorf("invalid start time format: %w", err)
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		return fmt.Errorf("invalid end time format: %w", err)
	}

	// Check if the new period overlaps with existing ones
	var existingRatios []database.InsulinRatio
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&existingRatios).Error; err != nil {
		return fmt.Errorf("failed to check existing ratios: %w", err)
	}

	// Convert times to minutes for easier comparison
	startMinutes := timeToMinutes(startTime)
	endMinutes := timeToMinutes(endTime)

	for _, r := range existingRatios {
		existingStart := timeToMinutes(r.StartTime)
		existingEnd := timeToMinutes(r.EndTime)

		if (startMinutes >= existingStart && startMinutes < existingEnd) ||
			(endMinutes > existingStart && endMinutes <= existingEnd) ||
			(startMinutes <= existingStart && endMinutes >= existingEnd) {
			return fmt.Errorf("time period overlaps with existing ratio")
		}
	}

	// Check if total coverage is 24 hours
	totalMinutes := 0
	for _, r := range existingRatios {
		existingStart := timeToMinutes(r.StartTime)
		existingEnd := timeToMinutes(r.EndTime)
		if existingEnd < existingStart {
			existingEnd += 24 * 60 // Add 24 hours if period crosses midnight
		}
		totalMinutes += existingEnd - existingStart
	}

	// Add new period
	if endMinutes < startMinutes {
		endMinutes += 24 * 60 // Add 24 hours if period crosses midnight
	}
	totalMinutes += endMinutes - startMinutes

	if totalMinutes > 24*60 {
		return fmt.Errorf("total time coverage exceeds 24 hours")
	}

	insulinRatio := &database.InsulinRatio{
		UserID:    userID,
		StartTime: startTime,
		EndTime:   endTime,
		Ratio:     ratio,
	}

	if err := s.db.WithContext(ctx).Create(insulinRatio).Error; err != nil {
		return fmt.Errorf("failed to create insulin ratio: %w", err)
	}

	return nil
}

func (s *InsulinService) GetUserRatios(ctx context.Context, userID uint) ([]database.InsulinRatio, error) {
	var ratios []database.InsulinRatio
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("start_time ASC").
		Find(&ratios).Error; err != nil {
		return nil, fmt.Errorf("failed to get user insulin ratios: %w", err)
	}
	return ratios, nil
}

func (s *InsulinService) DeleteRatio(ctx context.Context, userID uint, ratioID uint) error {
	result := s.db.WithContext(ctx).
		Where("user_id = ? AND id = ?", userID, ratioID).
		Delete(&database.InsulinRatio{})

	if result.Error != nil {
		return fmt.Errorf("failed to delete insulin ratio: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insulin ratio not found")
	}
	return nil
}

func (s *InsulinService) UpdateRatio(ctx context.Context, userID uint, ratioID uint, startTime, endTime string, ratio float64) error {
	// Validate time format
	if _, err := time.Parse("15:04", startTime); err != nil {
		return fmt.Errorf("invalid start time format: %w", err)
	}
	if _, err := time.Parse("15:04", endTime); err != nil {
		return fmt.Errorf("invalid end time format: %w", err)
	}

	// Check if the new period overlaps with existing ones (excluding the current ratio)
	var existingRatios []database.InsulinRatio
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND id != ?", userID, ratioID).
		Find(&existingRatios).Error; err != nil {
		return fmt.Errorf("failed to check existing ratios: %w", err)
	}

	// Convert times to minutes for easier comparison
	startMinutes := timeToMinutes(startTime)
	endMinutes := timeToMinutes(endTime)

	// Handle midnight crossing
	if endMinutes < startMinutes {
		endMinutes += 24 * 60
	}

	for _, r := range existingRatios {
		existingStart := timeToMinutes(r.StartTime)
		existingEnd := timeToMinutes(r.EndTime)

		// Handle midnight crossing for existing period
		if existingEnd < existingStart {
			existingEnd += 24 * 60
		}

		// Check for overlap
		if (startMinutes >= existingStart && startMinutes < existingEnd) ||
			(endMinutes > existingStart && endMinutes <= existingEnd) ||
			(startMinutes <= existingStart && endMinutes >= existingEnd) {
			return fmt.Errorf("time period overlaps with existing ratio")
		}
	}

	// Check if total coverage is 24 hours
	totalMinutes := 0
	for _, r := range existingRatios {
		existingStart := timeToMinutes(r.StartTime)
		existingEnd := timeToMinutes(r.EndTime)
		if existingEnd < existingStart {
			existingEnd += 24 * 60 // Add 24 hours if period crosses midnight
		}
		totalMinutes += existingEnd - existingStart
	}

	// Add new period
	totalMinutes += endMinutes - startMinutes

	if totalMinutes > 24*60 {
		return fmt.Errorf("total time coverage exceeds 24 hours")
	}

	result := s.db.WithContext(ctx).
		Model(&database.InsulinRatio{}).
		Where("user_id = ? AND id = ?", userID, ratioID).
		Updates(map[string]interface{}{
			"start_time": startTime,
			"end_time":   endTime,
			"ratio":      ratio,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to update insulin ratio: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("insulin ratio not found")
	}

	return nil
}

// Helper function to convert time string to minutes since midnight
func timeToMinutes(timeStr string) int {
	t, _ := time.Parse("15:04", timeStr)
	return t.Hour()*60 + t.Minute()
}

// GetActiveInsulinTime returns the active insulin time in minutes for a user
func (s *InsulinService) GetActiveInsulinTime(ctx context.Context, userID uint) (int, error) {
	var user database.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	return user.ActiveInsulinTime, nil
}

// SetActiveInsulinTime sets the active insulin time in minutes for a user
func (s *InsulinService) SetActiveInsulinTime(ctx context.Context, userID uint, minutes int) error {
	if err := s.db.Model(&database.User{}).Where("id = ?", userID).Update("active_insulin_time", minutes).Error; err != nil {
		return fmt.Errorf("failed to update active insulin time: %w", err)
	}
	return nil
}
