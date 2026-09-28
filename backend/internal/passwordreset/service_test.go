package passwordreset

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRepository struct {
	users             map[string]uuid.UUID
	tokens            map[string]mockToken
	rateLimitAllowed  bool
	consumedHashes    []string
	newPasswordHashes []string
}

type mockToken struct {
	userID    uuid.UUID
	expiresAt time.Time
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
	id, ok := m.users[strings.ToLower(strings.TrimSpace(email))]
	return id, ok, nil
}

func (m *mockRepository) CreateResetToken(_ context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time, _ time.Time) error {
	m.tokens[fmt.Sprintf("%x", tokenHash)] = mockToken{
		userID:    userID,
		expiresAt: expiresAt,
		used:      false,
	}
	return nil
}

func (m *mockRepository) ConsumeResetTokenAndChangePassword(_ context.Context, tokenHash []byte, consumedAt time.Time, hashPassword func() (string, error)) error {
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
	require.Len(t, repo.tokens, 1)

	// 2. Non-existing user produces identical nil error (no leak)
	delivery.Reset()
	errNonExisting := service.RequestReset(ctx, "not-found@tallerflow.pe", "127.0.0.1")
	require.NoError(t, errNonExisting)
	require.Empty(t, delivery.Deliveries(), "delivery must NOT send email to unregistered address")
	require.Len(t, repo.tokens, 1, "no token created for unregistered address")
}

func TestRequestResetRateLimiting(t *testing.T) {
	repo := newMockRepository()
	repo.rateLimitAllowed = false // simulate rate limit triggered
	delivery := NewMemoryDelivery()
	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{}, nil)

	err := service.RequestReset(context.Background(), "user@example.com", "127.0.0.1")
	require.ErrorIs(t, err, ErrRateLimited)
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
