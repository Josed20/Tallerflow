package analytics

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests require V20/V21 applied and the real restricted app role. The
// admin connection only creates/inspects/deletes this test's UUID-scoped graph;
// all event writes exercise Service.Record or the actual SQL function as app.
func TestPostgresIntegrationAnalyticsEventsEvidenceAndIsolation(t *testing.T) {
	appURL, adminURL := os.Getenv("TF_TEST_DATABASE_URL"), os.Getenv("TF_TEST_ADMIN_DATABASE_URL")
	if appURL == "" || adminURL == "" {
		t.Skip("TF_TEST_DATABASE_URL and TF_TEST_ADMIN_DATABASE_URL are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	db, err := platformdb.Open(appURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })
	admin, err := platformdb.Open(adminURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(admin)) })
	tenant, err := platformdb.NewTenantRunner(db)
	require.NoError(t, err)
	service := &Service{DB: db, Tenants: tenant}

	var bypass bool
	require.NoError(t, db.WithContext(ctx).Raw("SELECT rolsuper OR rolbypassrls FROM pg_roles WHERE rolname = current_user").Row().Scan(&bypass))
	require.False(t, bypass, "TF_TEST_DATABASE_URL must not bypass the RLS being tested")
	var forced bool
	require.NoError(t, admin.WithContext(ctx).Raw("SELECT relrowsecurity AND relforcerowsecurity FROM pg_class WHERE oid = 'analytics_events'::regclass").Row().Scan(&forced))
	require.True(t, forced)
	var now time.Time
	require.NoError(t, admin.WithContext(ctx).Raw("SELECT now()").Row().Scan(&now))
	verifiedAt := now.Add(-time.Minute)

	w, other := uuid.New(), uuid.New()
	owner, otherOwner, unverified, inactive, operator := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	anonymousInstallation := uuid.New()
	// Register cleanup before seeding, including for failed assertions/setup.
	t.Cleanup(func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 15*time.Second)
		defer stop()
		cleanup := func(query string, args ...any) {
			if e := admin.WithContext(cleanupCtx).Exec(query, args...).Error; e != nil {
				t.Errorf("analytics fixture cleanup: %v", e)
			}
		}
		cleanup("DELETE FROM analytics_events WHERE workshop_id IN (?, ?) OR installation_id = ?", w, other, anonymousInstallation)
		cleanup("DELETE FROM audit_events WHERE workshop_id IN (?, ?)", w, other)
		cleanup("DELETE FROM workshops WHERE id IN (?, ?)", w, other)
		cleanup("DELETE FROM users WHERE id IN (?, ?, ?, ?, ?)", owner, otherOwner, unverified, inactive, operator)
	})
	exec := func(t *testing.T, query string, args ...any) {
		t.Helper()
		require.NoError(t, admin.WithContext(ctx).Exec(query, args...).Error)
	}
	for _, id := range []uuid.UUID{owner, otherOwner, inactive, operator} {
		exec(t, "INSERT INTO users (id, email, name, created_at, email_verified_at) VALUES (?, ?, 'Analytics fixture', ?, ?)", id, id.String()+"@analytics-integration.test", now.Add(-48*time.Hour), verifiedAt)
	}
	exec(t, "INSERT INTO users (id, email, name, created_at) VALUES (?, ?, 'Unverified fixture', ?)", unverified, unverified.String()+"@analytics-integration.test", now.Add(-48*time.Hour))
	exec(t, "INSERT INTO workshops (id, name) VALUES (?, 'Analytics fixture A'), (?, 'Analytics fixture B')", w, other)
	exec(t, `INSERT INTO workshop_members (workshop_id, user_id, role, status) VALUES
		(?, ?, 'OWNER', 'ACTIVE'), (?, ?, 'OWNER', 'ACTIVE'),
		(?, ?, 'OWNER', 'ACTIVE'), (?, ?, 'OPERATOR', 'INACTIVE'), (?, ?, 'OPERATOR', 'ACTIVE')`,
		w, owner, other, otherOwner, w, unverified, w, inactive, w, operator)
	principal := httpx.Principal{WorkshopID: w, UserID: owner, Role: "OWNER"}
	otherPrincipal := httpx.Principal{WorkshopID: other, UserID: otherOwner, Role: "OWNER"}
	newEvent := func(kind string) Event {
		return Event{ID: uuid.New(), InstallationID: uuid.New(), Type: kind, OccurredAt: now, Channel: "android", Version: "3"}
	}
	count := func(t *testing.T, where string, args ...any) int64 {
		t.Helper()
		var result int64
		require.NoError(t, admin.WithContext(ctx).Raw("SELECT count(*) FROM analytics_events WHERE "+where, args...).Row().Scan(&result))
		return result
	}
	type storedEvent struct {
		ID            uuid.UUID
		WorkshopID    *uuid.UUID
		UserID        *uuid.UUID
		SourceEventID *uuid.UUID
		OccurredAt    time.Time
	}
	stored := func(t *testing.T, id uuid.UUID) storedEvent {
		t.Helper()
		var result storedEvent
		row := admin.WithContext(ctx).Raw("SELECT id, workshop_id, user_id, source_event_id, occurred_at FROM analytics_events WHERE id = ?", id).Scan(&result)
		require.NoError(t, row.Error)
		require.EqualValues(t, 1, row.RowsAffected)
		return result
	}

	t.Run("anonymous first open deduplicates event and installation without attribution", func(t *testing.T) {
		event := newEvent("first_open")
		event.InstallationID = anonymousInstallation
		require.NoError(t, service.Record(ctx, event, nil))
		require.NoError(t, service.Record(ctx, event, nil), "retry the same event UUID")
		duplicate := event
		duplicate.ID = uuid.New()
		require.NoError(t, service.Record(ctx, duplicate, nil), "another event UUID cannot count this installation twice")
		require.EqualValues(t, 1, count(t, "installation_id = ?", anonymousInstallation))
		require.Zero(t, count(t, "id = ?", duplicate.ID))
		row := stored(t, event.ID)
		require.Nil(t, row.WorkshopID)
		require.Nil(t, row.UserID)
		require.Nil(t, row.SourceEventID)
	})

	t.Run("activity rejects foreign unknown and inactive actors", func(t *testing.T) {
		for _, actor := range []uuid.UUID{otherOwner, uuid.New(), inactive} {
			forged := principal
			forged.UserID = actor
			event := newEvent("activity")
			require.ErrorIs(t, service.Record(ctx, event, &forged), ErrEvidenceInvalid)
			require.Zero(t, count(t, "id = ?", event.ID))
		}
		// Even a valid B actor cannot ask the SECURITY DEFINER function to write
		// into B while the transaction is explicitly scoped to A.
		event := newEvent("activity")
		require.NoError(t, tenant.WithinTenant(ctx, w, func(tx *gorm.DB) error {
			var accepted bool
			err := tx.Raw("SELECT analytics_record_event(?,?,?,?,?,?,?,?)", event.ID, event.InstallationID, event.Type, event.OccurredAt, event.Channel, event.Version, other, otherOwner).Row().Scan(&accepted)
			require.NoError(t, err)
			require.False(t, accepted)
			return nil
		}))
		require.Zero(t, count(t, "id = ?", event.ID))
		// A pooled connection without tenant context must fail closed too.
		require.NoError(t, db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var accepted bool
			err := tx.Raw("SELECT analytics_record_event(?,?,?,?,?,?,?,?)", event.ID, event.InstallationID, event.Type, event.OccurredAt, event.Channel, event.Version, w, owner).Row().Scan(&accepted)
			require.NoError(t, err)
			require.False(t, accepted)
			return nil
		}))
		require.Zero(t, count(t, "id = ?", event.ID))
	})

	t.Run("app reads stay tenant scoped and direct insert is forbidden", func(t *testing.T) {
		a, b := newEvent("activity"), newEvent("activity")
		require.NoError(t, service.Record(ctx, a, &principal))
		require.NoError(t, service.Record(ctx, a, &principal))
		require.NoError(t, service.Record(ctx, b, &otherPrincipal))
		require.EqualValues(t, 1, count(t, "id = ?", a.ID))
		for _, workshop := range []uuid.UUID{w, other} {
			require.NoError(t, tenant.WithinTenant(ctx, workshop, func(tx *gorm.DB) error {
				var visible []storedEvent
				if err := tx.Raw("SELECT id, workshop_id FROM analytics_events WHERE workshop_id IN (?, ?) OR installation_id = ?", w, other, anonymousInstallation).Scan(&visible).Error; err != nil {
					return err
				}
				require.Len(t, visible, 1, "neither another tenant nor anonymous telemetry may leak")
				require.NotNil(t, visible[0].WorkshopID)
				require.Equal(t, workshop, *visible[0].WorkshopID)
				return nil
			}))
		}
		var visible int64
		require.NoError(t, db.WithContext(ctx).Raw("SELECT count(*) FROM analytics_events WHERE workshop_id IN (?, ?) OR installation_id = ?", w, other, anonymousInstallation).Row().Scan(&visible))
		require.Zero(t, visible, "tenant context must not survive a completed transaction")
		injected := newEvent("activity")
		err := tenant.WithinTenant(ctx, w, func(tx *gorm.DB) error {
			return tx.Exec(`INSERT INTO analytics_events (id, installation_id, event_type, occurred_at, channel, version, workshop_id, user_id)
				VALUES (?, ?, 'activity', ?, 'android', '3', ?, ?)`, injected.ID, injected.InstallationID, now, w, owner).Error
		})
		var denied *pgconn.PgError
		require.ErrorAs(t, err, &denied)
		require.Equal(t, "42501", denied.Code, "app must write through the reviewed SQL function")
		require.Zero(t, count(t, "id = ?", injected.ID))
	})

	t.Run("registration needs verified owner evidence and counts once per workshop", func(t *testing.T) {
		for _, actor := range []uuid.UUID{unverified, operator} {
			p := principal
			p.UserID = actor
			event := newEvent("registration")
			require.ErrorIs(t, service.Record(ctx, event, &p), ErrEvidenceInvalid)
			require.Zero(t, count(t, "id = ?", event.ID))
		}
		event := newEvent("registration")
		require.NoError(t, service.Record(ctx, event, &principal))
		require.NoError(t, service.Record(ctx, event, &principal))
		require.NoError(t, service.Record(ctx, newEvent("registration"), &principal))
		require.EqualValues(t, 1, count(t, "workshop_id = ? AND event_type = 'registration'", w))
		row := stored(t, event.ID)
		require.True(t, row.OccurredAt.Equal(verifiedAt), "registration time must come from server verification evidence")
		require.NotNil(t, row.UserID)
		require.NotNil(t, row.WorkshopID)
		require.Equal(t, owner, *row.UserID)
		require.Equal(t, w, *row.WorkshopID)
	})

	t.Run("login requires matching channel evidence and consumes each sign in once", func(t *testing.T) {
		require.ErrorIs(t, service.Record(ctx, newEvent("login"), &otherPrincipal), ErrEvidenceInvalid)
		earlyID, lateID := uuid.New(), uuid.New()
		earlyTime, lateTime := now.Add(-2*time.Minute), now.Add(-time.Minute)
		exec(t, `INSERT INTO audit_events (id, workshop_id, actor_user_id, event_type, details, created_at) VALUES
			(?, ?, ?, 'USER_SIGNED_IN', '{"client":"mobile"}', ?),
			(?, ?, ?, 'USER_SIGNED_IN', '{"client":"mobile"}', ?)`, earlyID, w, owner, earlyTime, lateID, w, owner, lateTime)
		web := newEvent("login")
		web.Channel = "web"
		require.ErrorIs(t, service.Record(ctx, web, &principal), ErrEvidenceInvalid)
		require.Zero(t, count(t, "id = ?", web.ID))
		// Both client timestamps prefer the latest audit. The second event must
		// use the remaining audit instead of silently losing a genuine sign-in.
		first, second := newEvent("login"), newEvent("login")
		require.NoError(t, service.Record(ctx, first, &principal))
		require.NoError(t, service.Record(ctx, first, &principal), "same UUID remains retryable after its evidence is consumed")
		require.NoError(t, service.Record(ctx, second, &principal))
		firstRow, secondRow := stored(t, first.ID), stored(t, second.ID)
		require.NotNil(t, firstRow.SourceEventID)
		require.NotNil(t, secondRow.SourceEventID)
		require.Equal(t, lateID, *firstRow.SourceEventID)
		require.Equal(t, earlyID, *secondRow.SourceEventID)
		require.True(t, firstRow.OccurredAt.Equal(lateTime))
		require.True(t, secondRow.OccurredAt.Equal(earlyTime))
		require.EqualValues(t, 2, count(t, "workshop_id = ? AND event_type = 'login'", w))
		replay := newEvent("login")
		require.ErrorIs(t, service.Record(ctx, replay, &principal), ErrEvidenceInvalid)
		require.Zero(t, count(t, "id = ?", replay.ID))
	})

	t.Run("concurrent login events cannot reuse one server audit", func(t *testing.T) {
		evidenceID := uuid.New()
		exec(t, "INSERT INTO audit_events (id, workshop_id, actor_user_id, event_type, details, created_at) VALUES (?, ?, ?, 'USER_SIGNED_IN', '{\"client\":\"mobile\"}', ?)", evidenceID, other, otherOwner, verifiedAt)
		results := make(chan error, 2)
		start := make(chan struct{})
		var workers sync.WaitGroup
		for range 2 {
			event := newEvent("login")
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				results <- service.Record(ctx, event, &otherPrincipal)
			}()
		}
		close(start)
		workers.Wait()
		close(results)
		var accepted, rejected int
		for err := range results {
			switch {
			case err == nil:
				accepted++
			case errors.Is(err, ErrEvidenceInvalid):
				rejected++
			default:
				t.Fatalf("concurrent login: %v", err)
			}
		}
		require.Equal(t, 1, accepted)
		require.Equal(t, 1, rejected)
		require.EqualValues(t, 1, count(t, "source_event_id = ?", evidenceID))
	})
}
