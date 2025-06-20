package migrations

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Migration represents a database migration
type Migration struct {
	ID   string
	Up   func(*gorm.DB) error
	Down func(*gorm.DB) error
}

var migrations = make(map[string]Migration)

// Register adds a new migration to the registry
func Register(id string, up, down func(*gorm.DB) error) {
	migrations[id] = Migration{
		ID:   id,
		Up:   up,
		Down: down,
	}
}

// RunMigrations executes all pending migrations
func RunMigrations(db *gorm.DB) error {
	// Create migrations table with raw SQL instead of AutoMigrate
	if err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migration_records (
			id VARCHAR(255) PRIMARY KEY,
			created_at BIGINT DEFAULT EXTRACT(EPOCH FROM NOW())
		)
	`).Error; err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get all migration IDs
	var ids []string
	for id := range migrations {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Get executed migrations
	var executed []MigrationRecord
	if err := db.Find(&executed).Error; err != nil {
		return fmt.Errorf("failed to get executed migrations: %w", err)
	}

	executedMap := make(map[string]bool)
	for _, m := range executed {
		executedMap[m.ID] = true
	}

	// Run pending migrations
	for _, id := range ids {
		if !executedMap[id] {
			migration := migrations[id]
			log.Printf("Running migration: %s", id)
			if err := migration.Up(db); err != nil {
				return fmt.Errorf("failed to run migration %s: %w", id, err)
			}

			record := MigrationRecord{ID: id}
			if err := db.Create(&record).Error; err != nil {
				return fmt.Errorf("failed to record migration %s: %w", id, err)
			}
			log.Printf("Completed migration: %s", id)
		}
	}

	return nil
}

// MigrationRecord represents a record of executed migrations
type MigrationRecord struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt int64  `gorm:"autoCreateTime"`
}

// TableName returns the table name for this model
func (MigrationRecord) TableName() string {
	return "migration_records"
}

// LoadSQLMigrations loads SQL migrations from a directory
func LoadSQLMigrations(db *gorm.DB, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			id := strings.TrimSuffix(file.Name(), ".sql")
			path := filepath.Join(dir, file.Name())

			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %w", file.Name(), err)
			}

			// Register the migration
			Register(id, func(db *gorm.DB) error {
				if err := db.Exec(string(content)).Error; err != nil {
					// Ignore certain harmless errors in migrations
					if strings.Contains(err.Error(), "already exists") ||
						strings.Contains(err.Error(), "does not exist") {
						log.Printf("Migration %s: ignoring harmless error: %v", id, err)
						return nil
					}
					return err
				}
				return nil
			}, nil) // No down migration for SQL files
		}
	}

	return nil
}
