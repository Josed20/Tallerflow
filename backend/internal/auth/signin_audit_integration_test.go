package auth

import (
	"context"
	"os"
	"testing"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPostgresIntegrationSignInAuditStaysInItsWorkshop(t *testing.T) {
	databaseURL := os.Getenv("TF_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TF_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	db, err := platformdb.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })
	tenantRunner, err := platformdb.NewTenantRunner(db)
	require.NoError(t, err)
	audit, err := NewTenantSignInAudit(tenantRunner)
	require.NoError(t, err)

	userID, workshopID, otherWorkshopID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.WithContext(ctx).Exec(
		`INSERT INTO users (id, email, name, status, email_verified_at) VALUES (?, ?, 'Operario', 'ACTIVE', now())`, userID, "signin-"+userID.String()+"@example.com",
	).Error)
	require.NoError(t, seedTenant(ctx, tenantRunner, workshopID, userID, true))
	require.NoError(t, seedTenant(ctx, tenantRunner, otherWorkshopID, userID, false))

	require.NoError(t, audit.RecordSignIn(ctx, workshopID, userID, SessionKindMobile))
	require.NoError(t, audit.RecordSignIn(ctx, workshopID, userID, SessionKindWeb))

	clients := func(tenant uuid.UUID) []string {
		var found []string
		require.NoError(t, tenantRunner.WithinTenant(ctx, tenant, func(tx *gorm.DB) error {
			return tx.Raw(`SELECT details->>'client' FROM audit_events WHERE event_type = 'USER_SIGNED_IN' AND actor_user_id = ? ORDER BY created_at, details->>'client' DESC`, userID).Scan(&found).Error
		}))
		return found
	}
	require.ElementsMatch(t, []string{"mobile", "web"}, clients(workshopID))
	require.Empty(t, clients(otherWorkshopID), "another workshop must not see these sign-ins")
}
