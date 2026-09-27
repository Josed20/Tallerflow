package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"sync"
	"time"
)

type MemoryRateLimiter struct {
	mu       sync.Mutex
	secret   []byte
	clock    func() time.Time
	window   time.Duration
	limit    int
	capacity int
	entries  map[[32]byte]rateEntry
}

type rateEntry struct {
	count int
	start time.Time
}

func NewMemoryRateLimiter(secret []byte, clock func() time.Time, window time.Duration, limit, capacity int) *MemoryRateLimiter {
	if clock == nil {
		clock = time.Now
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if limit <= 0 {
		limit = 30
	}
	if capacity <= 0 {
		capacity = 10000
	}
	return &MemoryRateLimiter{
		secret: append([]byte(nil), secret...), clock: clock, window: window,
		limit: limit, capacity: capacity, entries: make(map[[32]byte]rateEntry),
	}
}

func (l *MemoryRateLimiter) Allow(key string) bool {
	if l == nil || len(l.secret) == 0 {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.clock().UTC()
	digest := l.digest(key)
	entry, exists := l.entries[digest]
	if exists && now.Sub(entry.start) >= l.window {
		delete(l.entries, digest)
		exists = false
	}
	if !exists {
		if len(l.entries) >= l.capacity {
			l.removeExpired(now)
		}
		if len(l.entries) >= l.capacity {
			return false
		}
		entry = rateEntry{start: now}
	}
	if entry.count >= l.limit {
		return false
	}
	entry.count++
	l.entries[digest] = entry
	return true
}

func (l *MemoryRateLimiter) removeExpired(now time.Time) {
	for key, entry := range l.entries {
		if now.Sub(entry.start) >= l.window {
			delete(l.entries, key)
		}
	}
}

func (l *MemoryRateLimiter) digest(key string) [32]byte {
	mac := hmac.New(sha256.New, l.secret)
	_, _ = mac.Write([]byte(key))
	var digest [32]byte
	copy(digest[:], mac.Sum(nil))
	return digest
}
