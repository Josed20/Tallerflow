package team

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrForbidden             = errors.New("team action is forbidden")
	ErrInvalidInput          = errors.New("team input is invalid")
	ErrInvitationUnavailable = errors.New("team invitation is unavailable")
	ErrDuplicateInvitation   = errors.New("team invitation already exists")
	ErrLastOwner             = errors.New("workshop must keep one active owner")
	ErrMemberNotFound        = errors.New("team member not found")
	ErrCrossWorkshopUser     = errors.New("user already belongs to another workshop")
	ErrInvitationRateLimited = errors.New("team invitation rate limited")
)

type Member struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Invitation struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

type InvitationCreated struct {
	Invitation Invitation `json:"invitation"`
	JoinURL    string     `json:"join_url"`
	Token      string     `json:"token"`
}

type InviteInput struct {
	Email string
	Role  string
}

type ConsumeInput struct {
	Token    string
	Name     string
	Password string
}

type UpdateMemberInput struct {
	Role   *string
	Status *string
}

type ConsumeResult struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
