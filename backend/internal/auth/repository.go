package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type SessionRepository interface {
	Insert(context.Context, NewSession) (Session, error)
	InsertForCredential(context.Context, string, NewSession) (Session, error)
	ChangePasswordAndInsert(context.Context, uuid.UUID, string, string, time.Time, NewSession) (Session, error)
	// FindActiveByTokenHash returns ErrSessionInvalid when no matching active
	// session exists, including unknown, expired, or revoked sessions. Database
	// adapters must translate their driver's no-row error to ErrSessionInvalid
	// and preserve operational errors so callers can distinguish outages.
	FindActiveByTokenHash(context.Context, [32]byte, time.Time) (Session, error)
	Revoke(context.Context, uuid.UUID, time.Time) error
	RevokeAllForUser(context.Context, uuid.UUID, time.Time) error
}
