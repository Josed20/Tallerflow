package onboarding

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPostgresIntegrationConcurrentOnboardingCreatesOneCompleteOwnerGraph(t *testing.T) {
	bootstrapURL := os.Getenv("TF_TEST_BOOTSTRAP_DATABASE_URL")
	appURL := os.Getenv("TF_TEST_DATABASE_URL")
	adminURL := os.Getenv("TF_TEST_ADMIN_DATABASE_URL")
	if bootstrapURL == "" || appURL == "" || adminURL == "" {
		t.Skip("TF_TEST_BOOTSTRAP_DATABASE_URL, TF_TEST_DATABASE_URL and TF_TEST_ADMIN_DATABASE_URL are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bootstrapPool, err := pgxpool.New(ctx, bootstrapURL)
	require.NoError(t, err)
	t.Cleanup(bootstrapPool.Close)
	adminPool, err := pgxpool.New(ctx, adminURL)
	require.NoError(t, err)
	t.Cleanup(adminPool.Close)
	appDB, err := platformdb.Open(appURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(appDB)) })
	require.NoError(t, bootstrapPool.Ping(ctx))
	require.NoError(t, adminPool.Ping(ctx))
	require.NoError(t, platformdb.Ping(ctx, appDB))
	require.Equal(t, int64(0), tableCount(t, ctx, adminPool, "bootstrap_state"), "integration database must start empty")

	now := time.Now().UTC().Truncate(time.Microsecond)
	repository, err := auth.NewPostgresRepository(appDB, func() time.Time { return now })
	require.NoError(t, err)
	sessions := auth.NewSessionService(repository, []byte("onboarding-integration-session-pepper"), func() time.Time { return now }, nil)
	bootstrap := auth.NewBootstrapServiceWithClock(
		auth.NewPostgresBootstrapStore(bootstrapPool),
		auth.NewPasswordHasher(auth.DefaultPasswordParams()),
		func() time.Time { return now },
	)
	service := NewService(
		NewPostgresStatusStore(bootstrapPool),
		OwnerCreatorFunc(func(ctx context.Context, input auth.BootstrapInput, metadata auth.SessionMetadata) (auth.WebBootstrapResult, error) {
			return bootstrap.CreateWebOwner(ctx, input, metadata, sessions)
		}),
	)

	inputs := []Input{
		{WorkshopName: "Taller Concurrente A", OwnerName: "Owner A", Email: "owner-a@integration.test", Password: "Secure integration password 123!", PasswordConfirmation: "Secure integration password 123!"},
		{WorkshopName: "Taller Concurrente B", OwnerName: "Owner B", Email: "owner-b@integration.test", Password: "Secure integration password 456!", PasswordConfirmation: "Secure integration password 456!"},
	}
	type outcome struct {
		result Result
		err    error
	}
	start := make(chan struct{})
	outcomes := make(chan outcome, len(inputs))
	var ready sync.WaitGroup
	ready.Add(len(inputs))
	for _, input := range inputs {
		go func(input Input) {
			ready.Done()
			<-start
			result, err := service.Create(ctx, input, auth.SessionMetadata{IPPrefix: "203.0.113.0/24", UserAgent: "integration-test"})
			outcomes <- outcome{result: result, err: err}
		}(input)
	}
	ready.Wait()
	close(start)

	var successes, conflicts int
	for range inputs {
		outcome := <-outcomes
		switch {
		case outcome.err == nil:
			successes++
			require.NotEmpty(t, outcome.result.Token)
			require.NotEmpty(t, outcome.result.CSRFToken)
			require.False(t, outcome.result.MustChangePassword)
		case errors.Is(outcome.err, ErrUnavailable):
			conflicts++
		default:
			require.NoError(t, outcome.err)
		}
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, conflicts)

	for _, table := range []string{"bootstrap_state", "users", "user_credentials", "workshops", "workshop_members", "audit_events", "user_sessions"} {
		require.Equal(t, int64(1), tableCount(t, ctx, adminPool, table), table)
	}
	status, err := service.Status(ctx)
	require.NoError(t, err)
	require.False(t, status.Available)
}

func tableCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, table string) int64 {
	t.Helper()
	allowed := map[string]bool{
		"bootstrap_state": true, "users": true, "user_credentials": true, "workshops": true,
		"workshop_members": true, "audit_events": true, "user_sessions": true,
	}
	require.True(t, allowed[table], "unsafe integration table name")
	var count int64
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count))
	return count
}
