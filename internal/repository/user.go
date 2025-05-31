package repository

import (
	"context"

	"github.com/vladimiradmaev/diabetes-helper/internal/domain"
	"gorm.io/gorm"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetOrCreateUser gets an existing user or creates a new one
func (r *UserRepository) GetOrCreateUser(ctx context.Context, telegramID int64, username, firstName, lastName string) (*domain.User, error) {
	var user domain.User
	result := r.db.Where("telegram_id = ?", telegramID).First(&user)
	if result.Error == nil {
		return &user, nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		return nil, result.Error
	}

	user = domain.User{
		TelegramID: telegramID,
		Username:   username,
		FirstName:  firstName,
		LastName:   lastName,
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByTelegramID gets a user by their Telegram ID
func (r *UserRepository) GetUserByTelegramID(ctx context.Context, telegramID int64) (*domain.User, error) {
	var user domain.User
	if err := r.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateActiveInsulinTime updates the active insulin time for a user
func (r *UserRepository) UpdateActiveInsulinTime(ctx context.Context, userID uint, minutes int) error {
	return r.db.Model(&domain.User{}).Where("id = ?", userID).Update("active_insulin_time", minutes).Error
}

// GetActiveInsulinTime gets the active insulin time for a user
func (r *UserRepository) GetActiveInsulinTime(ctx context.Context, userID uint) (int, error) {
	var user domain.User
	if err := r.db.First(&user, userID).Error; err != nil {
		return 0, err
	}
	return user.ActiveInsulinTime, nil
}
