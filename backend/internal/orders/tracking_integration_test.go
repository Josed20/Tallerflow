package orders

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPostgresIntegrationStableTrackingLifecycle(t *testing.T) {
	appURL, adminURL := os.Getenv("TF_TEST_DATABASE_URL"), os.Getenv("TF_TEST_ADMIN_DATABASE_URL")
	if appURL == "" || adminURL == "" {
		t.Skip("TF_TEST_DATABASE_URL and TF_TEST_ADMIN_DATABASE_URL are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := platformdb.Open(appURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })
	admin, err := platformdb.Open(adminURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(admin)) })
	tenant, err := platformdb.NewTenantRunner(db)
	require.NoError(t, err)
	r, err := NewPostgresTrackingRepository(db, tenant, "integration-only-tracking-secret")
	require.NoError(t, err)
	orderRepo, err := NewPostgresRepository(tenant)
	require.NoError(t, err)
	w, other, actor, client, orderID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(query string, args ...any) {
		t.Helper()
		require.NoError(t, admin.WithContext(ctx).Exec(query, args...).Error)
	}
	exec("INSERT INTO users (id, email, name) VALUES (?, ?, 'Tracking manager')", actor, actor.String()+"@tracking.test")
	exec("INSERT INTO workshops (id, name) VALUES (?, 'Tracking workshop'), (?, 'Other workshop')", w, other)
	exec("INSERT INTO clients (id, workshop_id, name) VALUES (?, ?, 'Tracking client')", client, w)
	exec("INSERT INTO orders (id, workshop_id, client_id, order_number, code, product, specification, quantity, unit, promised_at) VALUES (?, ?, ?, 1, 'OT-TRACK', 'Polos', 'Algodón azul', 12, 'UNITS', now())", orderID, w, client)
	t.Cleanup(func() {
		// Delete only this test's graph; do not disturb other integration fixtures.
		for _, q := range []string{"DELETE FROM audit_events WHERE workshop_id = ?", "DELETE FROM orders WHERE workshop_id = ?", "DELETE FROM clients WHERE workshop_id = ?", "DELETE FROM workshops WHERE id = ?"} {
			require.NoError(t, admin.Exec(q, w).Error)
		}
		require.NoError(t, admin.Exec("DELETE FROM workshops WHERE id = ?", other).Error)
		require.NoError(t, admin.Exec("DELETE FROM users WHERE id = ?", actor).Error)
	})
	exec("INSERT INTO order_stages (workshop_id, order_id, position, name, client_description, status) VALUES (?, ?, 1, 'Personalizada', 'Descripción para el cliente', 'IN_PROGRESS')", w, orderID)
	legacy := strings.Repeat("L", 43)
	exec("INSERT INTO order_tracking_links (workshop_id, order_id, token_hash, created_by) VALUES (?, ?, ?, ?)", w, orderID, trackingDigest(legacy), actor)

	// Concurrent recovery must return one stable token without revoking legacy.
	var wg sync.WaitGroup
	tokens := make(chan string, 6)
	errors := make(chan error, 6)
	for range 6 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token, e := r.ManageLink(ctx, w, orderID, actor, "recover")
			tokens <- token
			errors <- e
		}()
	}
	wg.Wait()
	close(tokens)
	close(errors)
	for e := range errors {
		require.NoError(t, e)
	}
	stable := <-tokens
	for token := range tokens {
		require.Equal(t, stable, token)
	}
	var stored struct {
		RecoveryNonce string
		TokenHash     []byte
	}
	require.NoError(t, admin.Raw("SELECT recovery_nonce, token_hash FROM order_tracking_links WHERE order_id = ? AND recovery_nonce IS NOT NULL AND revoked_at IS NULL", orderID).Scan(&stored).Error)
	require.NotEqual(t, stable, stored.RecoveryNonce, "the database must not store the bearer token")
	require.Equal(t, trackingDigest(stable), stored.TokenHash)
	for _, token := range []string{legacy, stable} {
		order, e := r.FindByToken(ctx, trackingDigest(token))
		require.NoError(t, e)
		require.Equal(t, "Algodón azul", order.Specification)
		require.Len(t, order.Stages, 1)
		require.Equal(t, "Descripción para el cliente", order.Stages[0].ClientDescription)
	}
	for _, action := range []string{"recover", "replace", "revoke"} {
		_, e := r.ManageLink(ctx, other, orderID, actor, action)
		require.ErrorIs(t, e, ErrOrderNotFound)
	}

	// Completion, unrelated edits, the exact expiry boundary and reopening.
	_, err = orderRepo.UpdateStage(ctx, w, orderID, 1, actor, "COMPLETED")
	require.NoError(t, err)
	var completed time.Time
	require.NoError(t, admin.Raw("SELECT tracking_completed_at FROM orders WHERE id = ?", orderID).Scan(&completed).Error)
	exec("UPDATE orders SET specification = 'Actualizado', updated_at = now() WHERE id = ?", orderID)
	var unchanged time.Time
	require.NoError(t, admin.Raw("SELECT tracking_completed_at FROM orders WHERE id = ?", orderID).Scan(&unchanged).Error)
	require.True(t, completed.Equal(unchanged))
	r.now = func() time.Time { return completed.Add(30*24*time.Hour - time.Microsecond) }
	_, err = r.FindByToken(ctx, trackingDigest(stable))
	require.NoError(t, err)
	r.now = func() time.Time { return completed.Add(30 * 24 * time.Hour) }
	for _, token := range []string{legacy, stable} {
		_, e := r.FindByToken(ctx, trackingDigest(token))
		require.ErrorIs(t, e, ErrTrackingNotFound)
	}
	_, err = orderRepo.ReopenStage(ctx, w, orderID, 1, actor, "Revisar entrega antes de cerrar")
	require.NoError(t, err)
	_, err = r.FindByToken(ctx, trackingDigest(stable))
	require.NoError(t, err)
	r.now = time.Now
	_, err = orderRepo.UpdateStage(ctx, w, orderID, 1, actor, "COMPLETED")
	require.NoError(t, err)
	var recompleted time.Time
	require.NoError(t, admin.Raw("SELECT tracking_completed_at FROM orders WHERE id = ?", orderID).Scan(&recompleted).Error)
	require.True(t, recompleted.After(completed))

	replacement, err := r.ManageLink(ctx, w, orderID, actor, "replace")
	require.NoError(t, err)
	require.NotEqual(t, stable, replacement)
	for _, token := range []string{legacy, stable} {
		_, e := r.FindByToken(ctx, trackingDigest(token))
		require.ErrorIs(t, e, ErrTrackingNotFound)
	}
	_, err = r.FindByToken(ctx, trackingDigest(replacement))
	require.NoError(t, err)
	_, err = r.ManageLink(ctx, w, orderID, actor, "revoke")
	require.NoError(t, err)
	_, err = r.FindByToken(ctx, trackingDigest(replacement))
	require.ErrorIs(t, err, ErrTrackingNotFound)
	recreated, err := r.ManageLink(ctx, w, orderID, actor, "recover")
	require.NoError(t, err)
	require.NotEqual(t, replacement, recreated)
	_, err = r.FindByToken(ctx, trackingDigest(replacement))
	require.ErrorIs(t, err, ErrTrackingNotFound)

	// A changed server secret must fail visibly instead of returning a dead URL.
	r.secret = []byte("a different server secret")
	_, err = r.ManageLink(ctx, w, orderID, actor, "recover")
	require.Error(t, err)
	_, err = r.FindByToken(ctx, trackingDigest(recreated))
	require.NoError(t, err)

	// Cancelling an expired completed order must preserve its original deadline.
	r.now = func() time.Time { return recompleted.Add(30 * 24 * time.Hour) }
	_, err = r.FindByToken(ctx, trackingDigest(recreated))
	require.ErrorIs(t, err, ErrTrackingNotFound)
	cancelled, err := orderRepo.Cancel(ctx, w, orderID, actor, "Cliente cancela el pedido completado")
	require.NoError(t, err)
	require.Equal(t, "CANCELLED", cancelled.Status)
	var cancelledCompletion time.Time
	require.NoError(t, admin.Raw("SELECT tracking_completed_at FROM orders WHERE id = ?", orderID).Scan(&cancelledCompletion).Error)
	require.True(t, recompleted.Equal(cancelledCompletion))
	_, err = r.FindByToken(ctx, trackingDigest(recreated))
	require.ErrorIs(t, err, ErrTrackingNotFound)
	// The actual correction API supports COMPLETED -> ACTIVE, not reopening a
	// cancelled order. Do not simulate an unsupported transition with direct SQL.
	_, err = orderRepo.ReopenStage(ctx, w, orderID, 1, actor, "Intento de reabrir pedido cancelado")
	require.ErrorIs(t, err, ErrOrderInvalid)
}
