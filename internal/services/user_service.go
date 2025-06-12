package services

import (
	"context"
	"fmt"

	"github.com/vladimiradmaev/diabetes-helper/internal/database"
	"gorm.io/gorm"
)

type UserService struct {
	db *gorm.DB
}

func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

func (s *UserService) RegisterUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*database.User, error) {
	// Try to find existing user first
	var user database.User
	result := s.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&user)

	if result.Error == nil {
		// User exists, return it
		return &user, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		// Some other error
		return nil, fmt.Errorf("failed to find user: %w", result.Error)
	}

	// User doesn't exist, create new one
	user = database.User{
		TelegramID: telegramID,
		Username:   username,
		FirstName:  firstName,
		LastName:   lastName,
	}

	if err := s.db.WithContext(ctx).Create(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return &user, nil
}

func (s *UserService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*database.User, error) {
	var user database.User
	if err := s.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}
