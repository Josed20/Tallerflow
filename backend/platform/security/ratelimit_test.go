package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimiterBlocksRotatingEmailsFromOneIP(t *testing.T) {
	now := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)
	limiter := NewMemoryLoginLimiter([]byte("test-secret"), func() time.Time { return now }, 15*time.Minute, 5)
	ctx := context.Background()
	for i, email := range []string{"one@example.com", "two@example.com", "three@example.com", "four@example.com", "five@example.com"} {
		allowed, err := limiter.Allow(ctx, email, "192.0.2.10")
		require.NoError(t, err)
		require.Truef(t, allowed, "attempt %d was blocked too early", i+1)
		require.NoError(t, limiter.RecordFailure(ctx, email, "192.0.2.10"))
	}

	allowed, err := limiter.Allow(ctx, "six@example.com", "192.0.2.10")

	require.NoError(t, err)
	require.False(t, allowed)
}

func TestLimiterSeparatesIndependentIPs(t *testing.T) {
	now := time.Date(2026, time.September, 26, 12, 0, 0, 0, time.UTC)
	limiter := NewMemoryLoginLimiter([]byte("test-secret"), func() time.Time { return now }, 15*time.Minute, 5)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, limiter.RecordFailure(ctx, "owner@example.com", "192.0.2.10"))
	}

	blocked, err := limiter.Allow(ctx, "new@example.com", "192.0.2.10")
	require.NoError(t, err)
	require.False(t, blocked)
	independent, err := limiter.Allow(ctx, "owner@example.com", "198.51.100.20")
	require.NoError(t, err)
	require.True(t, independent)
}
