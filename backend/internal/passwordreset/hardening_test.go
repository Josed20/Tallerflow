package passwordreset

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type countingHasher struct {
	calls int
}

func (h *countingHasher) Hash(password string) (string, error) {
	h.calls++
	return "hashed:" + password, nil
}

type failingDelivery struct{}

func (failingDelivery) Deliver(context.Context, string, string) error {
	return errors.New("smtp is unavailable")
}

func TestConsumeResetRejectsInvalidTokenBeforePasswordHashing(t *testing.T) {
	repo := newMockRepository()
	hasher := &countingHasher{}
	service := NewService(repo, NewMemoryDelivery(), hasher, ServiceConfig{}, nil)

	err := service.ConsumeReset(t.Context(), "invalid-token-012345678901234567890123456789", "ValidPassword123!")

	require.ErrorIs(t, err, ErrTokenInvalid)
	require.Zero(t, hasher.calls, "invalid tokens must not trigger password hashing")
}

func TestConsumeResetDoesNotHashExpiredOrUsedTokens(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	rawToken := "valid-token-012345678901234567890123456789"
	tokenHash, err := HashToken(rawToken)
	require.NoError(t, err)

	for name, token := range map[string]mockToken{
		"expired": {userID: uuid.New(), expiresAt: now.Add(-time.Minute)},
		"used":    {userID: uuid.New(), expiresAt: now.Add(time.Hour), used: true},
	} {
		t.Run(name, func(t *testing.T) {
			repo := newMockRepository()
			repo.tokens[fmt.Sprintf("%x", tokenHash)] = token
			hasher := &countingHasher{}
			service := NewService(repo, NewMemoryDelivery(), hasher, ServiceConfig{}, func() time.Time { return now })

			err := service.ConsumeReset(t.Context(), rawToken, "ValidPassword123!")

			if name == "expired" {
				require.ErrorIs(t, err, ErrTokenExpired)
			} else {
				require.ErrorIs(t, err, ErrTokenAlreadyUsed)
			}
			require.Zero(t, hasher.calls, "unusable tokens must not trigger password hashing")
		})
	}
}

func TestConsumeResetStopsBeforeHashingWhenRateLimited(t *testing.T) {
	repo := newMockRepository()
	repo.rateLimitAllowed = false
	hasher := &countingHasher{}
	service := NewService(repo, NewMemoryDelivery(), hasher, ServiceConfig{}, nil)

	err := service.ConsumeReset(t.Context(), "valid-token-012345678901234567890123456789", "ValidPassword123!", "127.0.0.1")

	require.ErrorIs(t, err, ErrRateLimited)
	require.Zero(t, hasher.calls)
}

func TestRequestResetHidesDeliveryFailureForRegisteredEmail(t *testing.T) {
	repo := newMockRepository()
	repo.users["owner@tallerflow.pe"] = uuid.New()
	service := NewService(repo, failingDelivery{}, mockHasher{}, ServiceConfig{}, nil)

	err := service.RequestReset(t.Context(), "owner@tallerflow.pe", "127.0.0.1")

	require.NoError(t, err, "a mail failure must not disclose that the address exists")
}
