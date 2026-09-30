package team

import (
	"context"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/google/uuid"
)

func TestConsumeRejectsInvalidProfileBeforeTokenLookup(t *testing.T) {
	service, err := NewService(&consumeStoreSpy{}, auth.NewPasswordHasher(auth.DefaultPasswordParams()), []byte("pepper"), "http://localhost:8080", nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Consume(context.Background(), ConsumeInput{
		Token:    "VHVKYWpXTllUSWVSc09KU2hwU3dPUUdPbG5IY3JvRGFGb0h6SWtfQ0hXdw",
		Name:     "Demo User",
		Password: "short",
	})

	if err != ErrInvalidInput {
		t.Fatalf("Consume() error = %v, want ErrInvalidInput", err)
	}
}

type consumeStoreSpy struct {
	Store
}

func TestRegenerateInvitationReplacesTheOneUseLink(t *testing.T) {
	invitationID := uuid.New()
	workshopID := uuid.New()
	actorID := uuid.New()
	now := time.Date(2026, time.September, 29, 10, 0, 0, 0, time.UTC)
	store := &regenerateStoreSpy{}
	service, err := NewService(store, auth.NewPasswordHasher(auth.DefaultPasswordParams()), []byte("pepper"), "http://localhost:8080", func() time.Time { return now }, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	created, err := service.RegenerateInvitation(context.Background(), httpx.Principal{
		WorkshopID: workshopID,
		UserID:     actorID,
		Role:       "OWNER",
	}, invitationID)
	if err != nil {
		t.Fatalf("RegenerateInvitation() error = %v", err)
	}
	if store.invitationID != invitationID || store.workshopID != workshopID || store.actorUserID != actorID {
		t.Fatal("RegenerateInvitation() did not scope the replacement to the current workshop and actor")
	}
	if len(store.tokenHash) == 0 || created.JoinURL == "" || created.Token == "" {
		t.Fatal("RegenerateInvitation() did not return a new one-use link")
	}
	if !store.expiresAt.Equal(now.Add(invitationLifetime)) {
		t.Fatalf("replacement expires at %s, want %s", store.expiresAt, now.Add(invitationLifetime))
	}
}

type regenerateStoreSpy struct {
	Store
	workshopID   uuid.UUID
	actorUserID  uuid.UUID
	invitationID uuid.UUID
	tokenHash    []byte
	expiresAt    time.Time
}

func (s *regenerateStoreSpy) RegenerateInvitation(_ context.Context, workshopID, actorUserID, invitationID uuid.UUID, tokenHash []byte, expiresAt, _ time.Time) (Invitation, error) {
	s.workshopID = workshopID
	s.actorUserID = actorUserID
	s.invitationID = invitationID
	s.tokenHash = append([]byte(nil), tokenHash...)
	s.expiresAt = expiresAt
	return Invitation{ID: invitationID, Email: "member@example.test", Role: "OPERATOR", ExpiresAt: expiresAt}, nil
}
