package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantWorkshopGate reads the workshop status inside that workshop's own
// tenant context: under RLS it is the only way the runtime role sees the row.
type TenantWorkshopGate struct {
	tenants database.TenantRunner
}

func NewTenantWorkshopGate(tenants database.TenantRunner) (*TenantWorkshopGate, error) {
	if tenants == nil {
		return nil, errors.New("tenant runner is required")
	}
	return &TenantWorkshopGate{tenants: tenants}, nil
}

// EnsureActive fails closed: a workshop that is not ACTIVE, or that the
// tenant context cannot see, is treated as suspended.
func (g *TenantWorkshopGate) EnsureActive(ctx context.Context, workshopID uuid.UUID) error {
	var status string
	err := g.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		result := tx.Raw(`SELECT status FROM workshops WHERE id = ?`, workshopID).Scan(&status)
		if result.Error != nil {
			return fmt.Errorf("read workshop status: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			status = ""
		}
		return nil
	})
	if err != nil {
		return err
	}
	if status != "ACTIVE" {
		return ErrWorkshopSuspended
	}
	return nil
}
