package database

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open creates the application's GORM connection. Schema changes are owned by
// Flyway; callers must never use GORM AutoMigrate.
func Open(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}
