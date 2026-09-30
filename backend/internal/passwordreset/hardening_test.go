package passwordreset

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
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

type signalingFailingDelivery struct {
	called chan struct{}
}

func (d signalingFailingDelivery) Deliver(context.Context, string, string) error {
	close(d.called)
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

func TestRequestResetKeepsPreviousTokenAndReportsDeliveryFailure(t *testing.T) {
	var logs synchronizedBuffer
	previousWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousWriter) })

	repo := newMockRepository()
	userID := uuid.New()
	repo.users["owner@tallerflow.pe"] = userID
	oldTokenKey := strings.Repeat("a", 64)
	repo.tokens[oldTokenKey] = mockToken{userID: userID, expiresAt: time.Now().Add(time.Hour)}
	delivery := signalingFailingDelivery{called: make(chan struct{})}
	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{}, nil)

	require.NoError(t, service.RequestReset(t.Context(), "owner@tallerflow.pe", "127.0.0.1"))
	select {
	case <-delivery.called:
	case <-time.After(time.Second):
		t.Fatal("delivery was not attempted")
	}

	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), "password reset delivery failed")
	}, time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool {
		return repo.hasToken(oldTokenKey) && repo.tokenCount() == 1
	}, time.Second, 10*time.Millisecond, "a failed delivery must not invalidate the previous usable token")
}

func TestRequestResetKeepsNewTokenUsableWhenOldTokenCleanupFails(t *testing.T) {
	var logs synchronizedBuffer
	previousWriter := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() { log.SetOutput(previousWriter) })

	repo := newMockRepository()
	userID := uuid.New()
	repo.users["owner@tallerflow.pe"] = userID
	repo.tokens[strings.Repeat("a", 64)] = mockToken{userID: userID, expiresAt: time.Now().Add(time.Hour)}
	repo.cleanupErr = errors.New("database unavailable")
	delivery := NewMemoryDelivery()
	service := NewService(repo, delivery, mockHasher{}, ServiceConfig{}, nil)

	require.NoError(t, service.RequestReset(t.Context(), "owner@tallerflow.pe", "127.0.0.1"))
	record := requireDelivery(t, delivery)
	rawToken := strings.Split(record.ResetURL, "token=")[1]
	newTokenHash, err := HashToken(rawToken)
	require.NoError(t, err)
	newTokenKey := fmt.Sprintf("%x", newTokenHash)

	require.Eventually(t, func() bool {
		return strings.Contains(logs.String(), "password reset token cleanup failed")
	}, time.Second, 10*time.Millisecond)
	require.True(t, repo.hasToken(newTokenKey), "a cleanup failure must not invalidate the delivered link")
}
