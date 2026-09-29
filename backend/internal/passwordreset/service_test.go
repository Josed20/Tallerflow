package passwordreset

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	mu                sync.Mutex
	users             map[string]uuid.UUID
	tokens            map[string]mockToken
	rateLimitAllowed  bool
	findErr           error
	createErr         error
	deleteErr         error
	cleanupErr        error
	consumedHashes    []string
	newPasswordHashes []string
}

type synchronizedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronizedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

type mockToken struct {
	userID    uuid.UUID
	expiresAt time.Time
	createdAt time.Time
	used      bool
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users:            make(map[string]uuid.UUID),
		tokens:           make(map[string]mockToken),
		rateLimitAllowed: true,
	}
}

func (m *mockRepository) FindUserByEmail(_ context.Context, email string) (uuid.UUID, bool, error) {
	if m.findErr != nil {
		return uuid.Nil, false, m.findErr
	}
	id, ok := m.users[strings.ToLower(strings.TrimSpace(email))]
	return id, ok, nil
}

func (m *mockRepository) CreateResetToken(_ context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time, createdAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.createErr != nil {
		return m.createErr
	}
	m.tokens[fmt.Sprintf("%x", tokenHash)] = mockToken{
		userID:    userID,
		expiresAt: expiresAt,
		createdAt: createdAt,
		used:      false,
	}
	return nil
}

func (m *mockRepository) DeleteResetToken(_ context.Context, tokenHash []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.tokens, fmt.Sprintf("%x", tokenHash))
	return nil
}

func (m *mockRepository) DeleteOlderResetTokens(_ context.Context, userID uuid.UUID, tokenHash []byte, createdAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.cleanupErr != nil {
		return m.cleanupErr
	}
	currentKey := fmt.Sprintf("%x", tokenHash)
	for key, token := range m.tokens {
		if token.userID != userID || token.used || key == currentKey {
			continue
		}
		if token.createdAt.Before(createdAt) || (token.createdAt.Equal(createdAt) && key < currentKey) {
			delete(m.tokens, key)
		}
	}
	return nil
}

func (m *mockRepository) ConsumeResetTokenAndChangePassword(_ context.Context, tokenHash []byte, consumedAt time.Time, hashPassword func() (string, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%x", tokenHash)
	token, ok := m.tokens[key]
	if !ok {
		return ErrTokenInvalid
	}
	if token.used {
		return ErrTokenAlreadyUsed
	}
	if !token.expiresAt.After(consumedAt) {
		return ErrTokenExpired
	}
	newPasswordHash, err := hashPassword()
	if err != nil {
		return err
	}
	token.used = true
	m.tokens[key] = token
	m.consumedHashes = append(m.consumedHashes, key)
	m.newPasswordHashes = append(m.newPasswordHashes, newPasswordHash)
	return nil
}

func (m *mockRepository) hasToken(key string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, exists := m.tokens[key]
	return exists
}

func (m *mockRepository) tokenCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.tokens)
}

func (m *mockRepository) AllowRequest(_ context.Context, _, _ string) (bool, error) {
	return m.rateLimitAllowed, nil
}

func (m *mockRepository) AllowConsume(_ context.Context, _ string, _ []byte) (bool, error) {
	return m.rateLimitAllowed, nil
}

func requireDelivery(t *testing.T, delivery *MemoryDelivery) DeliveryRecord {
	t.Helper()
	var record DeliveryRecord
	require.Eventually(t, func() bool {
		record, _ = delivery.LastDelivery()
		return record.ResetURL != ""
	}, time.Second, 10*time.Millisecond)
	return record
}

type mockHasher struct{}

func (m mockHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func TestRequestResetAntiEnumeration(t *testing.T) {
	repo := newMockRepository()
	existingUserID := uuid.New()
	repo.users["registered@tallerflow.pe"] = existingUserID

	delivery := NewMemoryDelivery()
	clock := func() time.Time { return time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC) }
	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{
		BaseURL:       "http://localhost:8080",
		TokenLifetime: 1 * time.Hour,
	}, clock)

	ctx := context.Background()

	// 1. Existing user
	errExisting := service.RequestReset(ctx, "registered@tallerflow.pe", "127.0.0.1")
	require.NoError(t, errExisting)
	record := requireDelivery(t, delivery)
	require.Equal(t, "registered@tallerflow.pe", record.RecipientEmail)
	require.Contains(t, record.ResetURL, "http://localhost:8080/reset-password?token=")
	require.Eventually(t, func() bool { return repo.tokenCount() == 1 }, time.Second, 10*time.Millisecond)

	// 2. Non-existing user produces identical nil error (no leak)
	delivery.Reset()
	errNonExisting := service.RequestReset(ctx, "not-found@tallerflow.pe", "127.0.0.1")
	require.NoError(t, errNonExisting)
	require.Empty(t, delivery.Deliveries(), "delivery must NOT send email to unregistered address")
	require.Equal(t, 1, repo.tokenCount(), "no token created for unregistered address")
}

func TestRequestResetRateLimiting(t *testing.T) {
	repo := newMockRepository()
	repo.rateLimitAllowed = false // simulate rate limit triggered
	delivery := NewMemoryDelivery()
	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{}, nil)

	err := service.RequestReset(context.Background(), "user@example.com", "127.0.0.1")
	require.ErrorIs(t, err, ErrRateLimited)
}

func TestRequestResetPropagatesOperationalFailuresWithoutExposingAccountExistence(t *testing.T) {
	t.Run("user lookup failure", func(t *testing.T) {
		repo := newMockRepository()
		repo.findErr = errors.New("database unavailable")
		service := NewService(repo, NewMemoryDelivery(), mockHasher{}, ServiceConfig{}, nil)

		err := service.RequestReset(t.Context(), "user@example.com", "127.0.0.1")

		require.Error(t, err)
		require.NotErrorIs(t, err, ErrRateLimited)
	})

	t.Run("token persistence failure", func(t *testing.T) {
		var logs synchronizedBuffer
		previousWriter := log.Writer()
		log.SetOutput(&logs)
		t.Cleanup(func() { log.SetOutput(previousWriter) })

		repo := newMockRepository()
		repo.users["user@example.com"] = uuid.New()
		repo.createErr = errors.New("database unavailable")
		service := NewService(repo, NewMemoryDelivery(), mockHasher{}, ServiceConfig{}, nil)

		err := service.RequestReset(t.Context(), "user@example.com", "127.0.0.1")

		require.NoError(t, err, "background persistence failures must not disclose account existence")
		require.Eventually(t, func() bool {
			return strings.Contains(logs.String(), "password reset token persistence failed")
		}, time.Second, 10*time.Millisecond)
	})
}

func TestConsumeReset(t *testing.T) {
	repo := newMockRepository()
	delivery := NewMemoryDelivery()
	currentTime := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return currentTime }

	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{
		BaseURL:       "http://localhost:8080",
		TokenLifetime: 1 * time.Hour,
	}, clock)

	ctx := context.Background()
	userID := uuid.New()
	repo.users["user@example.com"] = userID

	require.NoError(t, service.RequestReset(ctx, "user@example.com", "127.0.0.1"))
	last := requireDelivery(t, delivery)

	// Extract raw token from reset URL
	parts := strings.Split(last.ResetURL, "token=")
	require.Len(t, parts, 2)
	rawToken := parts[1]

	t.Run("rejects password shorter than 12 characters", func(t *testing.T) {
		err := service.ConsumeReset(ctx, rawToken, "shortpass")
		require.ErrorIs(t, err, ErrPasswordTooWeak)
	})

	t.Run("rejects invalid or altered raw token", func(t *testing.T) {
		err := service.ConsumeReset(ctx, "altered-token-012345678901234567890123456789", "ValidPassword1234!")
		require.ErrorIs(t, err, ErrTokenInvalid)
	})

	t.Run("successfully consumes token and changes password", func(t *testing.T) {
		err := service.ConsumeReset(ctx, rawToken, "MyNewSecurePassword2026!")
		require.NoError(t, err)
		require.Len(t, repo.consumedHashes, 1)
		require.Equal(t, "hashed:MyNewSecurePassword2026!", repo.newPasswordHashes[0])
	})

	t.Run("rejects consuming the token a second time", func(t *testing.T) {
		err := service.ConsumeReset(ctx, rawToken, "AnotherPassword1234!")
		require.ErrorIs(t, err, ErrTokenAlreadyUsed)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		// Advance clock past 1 hour expiry
		currentTime = currentTime.Add(2 * time.Hour)
		// Request a new token
		delivery.Reset()
		require.NoError(t, service.RequestReset(ctx, "user@example.com", "127.0.0.1"))
		newDeliv := requireDelivery(t, delivery)
		newToken := strings.Split(newDeliv.ResetURL, "token=")[1]

		// Now advance clock past expiry of this token as well
		currentTime = currentTime.Add(2 * time.Hour)
		err := service.ConsumeReset(ctx, newToken, "PasswordAfterExpiry123!")
		require.ErrorIs(t, err, ErrTokenExpired)
	})
}
