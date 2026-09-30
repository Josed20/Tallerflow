package team

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Store interface {
	ListMembers(ctx context.Context, workshopID uuid.UUID) ([]Member, error)
	ListInvitations(ctx context.Context, workshopID uuid.UUID, now time.Time) ([]Invitation, error)
	CreateInvitation(ctx context.Context, workshopID, actorUserID uuid.UUID, email, role string, tokenHash []byte, expiresAt time.Time) (Invitation, error)
	RegenerateInvitation(ctx context.Context, workshopID, actorUserID, invitationID uuid.UUID, tokenHash []byte, expiresAt, now time.Time) (Invitation, error)
	CancelInvitation(ctx context.Context, workshopID, actorUserID, invitationID uuid.UUID, now time.Time) error
	ConsumeInvitation(ctx context.Context, tokenHash []byte, name, passwordHash string, now time.Time) (ConsumeResult, error)
	UpdateMember(ctx context.Context, workshopID, actorUserID, membershipID uuid.UUID, input UpdateMemberInput) (Member, error)
}
