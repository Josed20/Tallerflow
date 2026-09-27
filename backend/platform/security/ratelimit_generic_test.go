package security

import (
	"testing"
	"time"
)

func TestMemoryRateLimiterEnforcesWindowPerOpaqueKey(t *testing.T) {
	now := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	limiter := NewMemoryRateLimiter([]byte("test-secret"), func() time.Time { return now }, time.Minute, 2, 10)

	if !limiter.Allow("203.0.113.10") || !limiter.Allow("203.0.113.10") {
		t.Fatal("limiter rejected a request below the configured limit")
	}
	if limiter.Allow("203.0.113.10") {
		t.Fatal("limiter allowed a request above the configured limit")
	}
	if !limiter.Allow("203.0.113.11") {
		t.Fatal("one key throttled an unrelated key")
	}

	now = now.Add(time.Minute)
	if !limiter.Allow("203.0.113.10") {
		t.Fatal("limiter did not reset after the fixed window")
	}
}

func TestMemoryRateLimiterFailsClosedAtBoundedCapacity(t *testing.T) {
	limiter := NewMemoryRateLimiter([]byte("test-secret"), time.Now, time.Hour, 5, 1)

	if !limiter.Allow("first") {
		t.Fatal("first key was rejected")
	}
	if limiter.Allow("second") {
		t.Fatal("limiter accepted an untracked key after reaching capacity")
	}
}
