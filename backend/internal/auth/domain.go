package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrSessionInvalid       = errors.New("session is invalid")
	ErrSessionConfiguration = errors.New("session service is not configured")
	ErrCredentialChanged    = errors.New("credential changed during authentication")
)

type SessionMetadata struct {
	IPPrefix  string
	UserAgent string
}

type NewSession struct {
	UserID        uuid.UUID
	TokenHash     [32]byte
	CSRFTokenHash [32]byte
	Metadata      SessionMetadata
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type RawSession struct {
	Token     string
	CSRFToken string
	ExpiresAt time.Time
	Session   Session
}

type ActiveMembership struct {
	WorkshopID uuid.UUID
	Role       string
}
