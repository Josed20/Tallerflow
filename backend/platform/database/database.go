package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Open creates the application's GORM connection. Schema changes are owned by
// Flyway; callers must never use GORM AutoMigrate.
func Open(databaseURL string) (*gorm.DB, error) {
	if databaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	db, err := gorm.Open(postgres.Open(databaseURL), runtimeGORMConfig())
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	return db, nil
}

func runtimeGORMConfig() *gorm.Config {
	safeLogger := gormlogger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: false,
		Colorful:                  false,
		ParameterizedQueries:      true,
	})
	return &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   safeLogger,
	}
}

// Ping checks the underlying database/sql connection used by GORM.
func Ping(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("ping database: database is required")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

// Close closes the underlying database/sql connection used by GORM.
func Close(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("close database: database is required")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}
	return nil
}
