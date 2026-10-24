package auth

import (
	"context"
	"os"
	"testing"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPostgresIntegrationWorkshopGateFollowsTheConsoleStatus(t *testing.T) {
	databaseURL, adminURL := os.Getenv("TF_TEST_DATABASE_URL"), os.Getenv("TF_TEST_ADMIN_DATABASE_URL")
	if databaseURL == "" || adminURL == "" {
		t.Skip("TF_TEST_DATABASE_URL and TF_TEST_ADMIN_DATABASE_URL are required")
	}
	ctx := context.Background()
	db, err := platformdb.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })
	// Only the console changes a workshop's status; the test stands in for it.
	admin, err := platformdb.Open(adminURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(admin)) })

	tenantRunner, err := platformdb.NewTenantRunner(db)
	require.NoError(t, err)
	gate, err := NewTenantWorkshopGate(tenantRunner)
	require.NoError(t, err)
	workshopID := uuid.New()
	require.NoError(t, seedTenant(ctx, tenantRunner, workshopID, uuid.Nil, false))
	setStatus := func(status string) {
		require.NoError(t, admin.WithContext(ctx).Exec(`UPDATE workshops SET status = ? WHERE id = ?`, status, workshopID).Error)
	}

	require.NoError(t, gate.EnsureActive(ctx, workshopID), "the runtime role must see its own workshop under RLS")
	setStatus("SUSPENDED")
	require.ErrorIs(t, gate.EnsureActive(ctx, workshopID), ErrWorkshopSuspended)
	setStatus("PENDING_DELETION")
	require.ErrorIs(t, gate.EnsureActive(ctx, workshopID), ErrWorkshopSuspended)
	setStatus("ACTIVE")
	require.NoError(t, gate.EnsureActive(ctx, workshopID))
	require.ErrorIs(t, gate.EnsureActive(ctx, uuid.New()), ErrWorkshopSuspended, "an unknown workshop fails closed")
}
