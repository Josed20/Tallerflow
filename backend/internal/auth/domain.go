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

// SessionKind separates browser sessions (cookie + CSRF, eight hours) from
// Android app sessions (bearer token, 30 days renewed with use). A token only
// authenticates through the transport of its own kind.
type SessionKind string

const (
	SessionKindWeb    SessionKind = "WEB"
	SessionKindMobile SessionKind = "MOBILE"
)

// normalized treats the zero value as WEB, so existing callers that never
// set a kind keep opening browser sessions.
func (k SessionKind) normalized() SessionKind {
	if k == SessionKindMobile {
		return SessionKindMobile
	}
	return SessionKindWeb
}

type SessionMetadata struct {
	IPPrefix  string
	UserAgent string
	Kind      SessionKind
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
	Kind      SessionKind
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
