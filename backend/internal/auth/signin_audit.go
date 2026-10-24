package auth

import (
	"context"
	"errors"
	"strings"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TenantSignInAudit writes each successful login to the workshop's audit
// trail as USER_SIGNED_IN, with the client ("web" or "mobile") that opened it.
type TenantSignInAudit struct {
	tenants database.TenantRunner
}

func NewTenantSignInAudit(tenants database.TenantRunner) (*TenantSignInAudit, error) {
	if tenants == nil {
		return nil, errors.New("tenant runner is required")
	}
	return &TenantSignInAudit{tenants: tenants}, nil
}

func (a *TenantSignInAudit) RecordSignIn(ctx context.Context, workshopID, userID uuid.UUID, kind SessionKind) error {
	client := strings.ToLower(string(kind.normalized()))
	return a.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Exec(
			`INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, 'USER_SIGNED_IN', jsonb_build_object('client', ?::text))`,
			workshopID, userID, client,
		).Error
	})
}
