package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrTenantRequired = errors.New("tenant workshop ID is required")

type TenantFunc func(tx *gorm.DB) error

// TenantRunner is the only supported entry point for tenant-owned data.
type TenantRunner interface {
	WithinTenant(ctx context.Context, workshopID uuid.UUID, fn TenantFunc) error
}

type GormTenantRunner struct {
	db *gorm.DB
}

func NewTenantRunner(db *gorm.DB) (*GormTenantRunner, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	return &GormTenantRunner{db: db}, nil
}

func (r *GormTenantRunner) WithinTenant(ctx context.Context, workshopID uuid.UUID, fn TenantFunc) error {
	if workshopID == uuid.Nil {
		return ErrTenantRequired
	}
	if fn == nil {
		return errors.New("tenant operation is required")
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The third argument makes the setting transaction-local, equivalent to
		// SET LOCAL app.workshop_id, while still allowing a bound parameter.
		if err := tx.Exec("SELECT set_config('app.workshop_id', ?, true)", workshopID.String()).Error; err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}
		return fn(tx)
	})
	if err != nil {
		return fmt.Errorf("tenant transaction: %w", err)
	}
	return nil
}
