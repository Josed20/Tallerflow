package passwordreset

import (
	"context"
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
