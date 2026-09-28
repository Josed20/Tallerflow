package passwordreset

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrTokenInvalid     = errors.New("recovery token is invalid")
	ErrTokenExpired     = errors.New("recovery token has expired")
	ErrTokenAlreadyUsed = errors.New("recovery token has already been used")
	ErrPasswordTooWeak  = errors.New("password does not meet security requirements")
	ErrRateLimited      = errors.New("too many password reset requests")
)

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
