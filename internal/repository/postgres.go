package repository

import (
	"fmt"
	"log"

	"github.com/vladimiradmaev/diabetes-helper/internal/config"
	"github.com/vladimiradmaev/diabetes-helper/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// PostgresDB represents a PostgreSQL database connection
type PostgresDB struct {
	db *gorm.DB
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg config.DBConfig) (*PostgresDB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.FoodAnalysis{},
		&domain.BloodSugarRecord{},
		&domain.InsulinRatio{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Println("Database connection established and migrations completed")
	return &PostgresDB{db: db}, nil
}

// GetDB returns the underlying GORM database instance
func (p *PostgresDB) GetDB() *gorm.DB {
	return p.db
}
