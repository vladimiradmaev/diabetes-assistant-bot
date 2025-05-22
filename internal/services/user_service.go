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
	user := &database.User{
		TelegramID: telegramID,
		Username:   username,
		FirstName:  firstName,
		LastName:   lastName,
	}

	result := s.db.WithContext(ctx).FirstOrCreate(user, database.User{TelegramID: telegramID})
	if result.Error != nil {
		return nil, fmt.Errorf("failed to register user: %w", result.Error)
	}

	return user, nil
}

func (s *UserService) GetUserByTelegramID(ctx context.Context, telegramID int64) (*database.User, error) {
	var user database.User
	if err := s.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return &user, nil
}
