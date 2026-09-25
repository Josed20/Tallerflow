package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"sync"
	"time"
)

// MemoryLoginLimiter is a development/single-process adapter. It loses history
// on restart; production should implement the auth port against login_attempts
// in PostgreSQL so every API instance sees the same bounded attempt history.
type MemoryLoginLimiter struct {
	mu          sync.Mutex
	secret      []byte
	now         func() time.Time
	window      time.Duration
	maxFailures int
	maxEntries  int
	attempts    map[[32]byte]loginAttempt
}

type loginAttempt struct {
	failures int
	first    time.Time
}

func NewMemoryLoginLimiter(secret []byte, now func() time.Time, window time.Duration, maxFailures int) *MemoryLoginLimiter {
	if now == nil {
		now = time.Now
	}
	if window <= 0 {
		window = 15 * time.Minute
	}
	if maxFailures <= 0 {
		maxFailures = 5
	}
	return &MemoryLoginLimiter{
		secret: append([]byte(nil), secret...), now: now, window: window,
		maxFailures: maxFailures, maxEntries: 10000, attempts: make(map[[32]byte]loginAttempt),
	}
}

func (l *MemoryLoginLimiter) Allow(_ context.Context, email, ip string) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := l.key(email, ip)
	entry, ok := l.attempts[key]
	if !ok || l.now().Sub(entry.first) >= l.window {
		delete(l.attempts, key)
		if !l.hasCapacity() {
			return false, nil
		}
		return true, nil
	}
	return entry.failures < l.maxFailures, nil
}

func (l *MemoryLoginLimiter) RecordFailure(_ context.Context, email, ip string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := l.key(email, ip)
	entry, ok := l.attempts[key]
	if !ok || l.now().Sub(entry.first) >= l.window {
		delete(l.attempts, key)
		if !l.hasCapacity() {
			return ErrLoginLimiterCapacity
		}
		entry = loginAttempt{first: l.now()}
	}
	entry.failures++
	l.attempts[key] = entry
	return nil
}

var ErrLoginLimiterCapacity = errors.New("login limiter capacity reached")

// hasCapacity is called with mu held. It removes expired entries only when
// needed and fails closed once every live slot is in use.
func (l *MemoryLoginLimiter) hasCapacity() bool {
	if len(l.attempts) < l.maxEntries {
		return true
	}
	now := l.now()
	for key, entry := range l.attempts {
		if now.Sub(entry.first) >= l.window {
			delete(l.attempts, key)
		}
	}
	return len(l.attempts) < l.maxEntries
}

func (l *MemoryLoginLimiter) key(email, ip string) [32]byte {
	mac := hmac.New(sha256.New, l.secret)
	_, _ = mac.Write([]byte(email))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(ip))
	var key [32]byte
	copy(key[:], mac.Sum(nil))
	return key
}
