package passwordreset

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPostgresIntegrationRecoveryRateLimitIsAtomicAcrossConcurrentRequests(t *testing.T) {
	databaseURL := os.Getenv("TF_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TF_TEST_DATABASE_URL is not configured")
	}
	db, err := platformdb.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })

	repository, err := NewPostgresRepository(db, time.Now)
	require.NoError(t, err)
	ip := "198.51.100." + uuid.NewString()
	var allowed atomic.Int32
	var group sync.WaitGroup
	errs := make(chan error, 12)
	for index := 0; index < 12; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			ok, err := repository.AllowRequest(context.Background(), ip, uuid.NewString()+"@example.com")
			if err != nil {
				errs <- err
				return
			}
			if ok {
				allowed.Add(1)
			}
		}(index)
	}
	group.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, int32(recoveryRateMaxAttempts), allowed.Load())
}

func TestPostgresIntegrationResetTokenIsSingleUseUnderConcurrentConsume(t *testing.T) {
	databaseURL := os.Getenv("TF_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TF_TEST_DATABASE_URL is not configured")
	}
	db, err := platformdb.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })

	repository, err := NewPostgresRepository(db, time.Now)
	require.NoError(t, err)
	now := time.Now().UTC()
	userID := uuid.New()
	tokenHash := sha256.Sum256([]byte(uuid.NewString()))
	require.NoError(t, db.Exec(`INSERT INTO users (id, email, name) VALUES (?, ?, ?)`, userID, uuid.NewString()+"@example.com", "Recovery Test").Error)
	require.NoError(t, db.Exec(`INSERT INTO user_credentials (user_id, password_hash) VALUES (?, ?)`, userID, "old-password-hash").Error)
	require.NoError(t, db.Exec(`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (gen_random_uuid(), ?, ?, ?, ?)`, userID, tokenHash[:], now.Add(time.Hour), now).Error)

	results := make(chan error, 2)
	start := make(chan struct{})
	var group sync.WaitGroup
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			results <- repository.ConsumeResetTokenAndChangePassword(context.Background(), tokenHash[:], now, func() (string, error) {
				return "new-password-hash", nil
			})
		}()
	}
	close(start)
	group.Wait()
	close(results)

	var successes int
	var alreadyUsed int
	for err := range results {
		if err == nil {
			successes++
			continue
		}
		if errors.Is(err, ErrTokenAlreadyUsed) {
			alreadyUsed++
			continue
		}
		require.NoError(t, err)
	}
	require.Equal(t, 1, successes)
	require.Equal(t, 1, alreadyUsed)
}
