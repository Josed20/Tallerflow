package signup

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrClosed       = errors.New("signup is closed")
	ErrRateLimited  = errors.New("too many signup requests")
	ErrTokenInvalid = errors.New("verification token is invalid, used or expired")
	// ErrPasswordMismatch means the link is fine but whoever opened it does not
	// know the password the account was registered with.
	ErrPasswordMismatch = errors.New("password does not match the registration")
)

type Input struct {
	WorkshopName string
	OwnerName    string
	Email        string
	Password     string
	AcceptTerms  bool
}

// ValidationError lists the request fields that failed, in the contract's
// field names, so the form can point at each one.
type ValidationError struct {
	Fields []string
}

func (e *ValidationError) Error() string { return "signup input is invalid" }

// PendingAccount is one registration attempt, already validated and hashed.
type PendingAccount struct {
	Email        string
	OwnerName    string
	WorkshopName string
	PasswordHash string
	TokenHash    []byte
	ExpiresAt    time.Time
}

// Recipient is who receives a verification email.
type Recipient struct {
	Name         string
	WorkshopName string
}

type Repository interface {
	// Register creates the owner, the ACTIVE workshop, the OWNER membership and
	// a pending verification token in one transaction. When the email belongs
	// to an account that was never verified, it replaces that account's name,
	// workshop name, password and token instead, so only the mailbox owner can
	// activate it. It returns issued=false, without error, when the email
	// belongs to a verified account.
	Register(context.Context, PendingAccount) (issued bool, err error)
	// ReissueToken replaces the pending token of an unverified account and
	// returns issued=false for unknown or already verified emails.
	ReissueToken(ctx context.Context, email string, tokenHash []byte, expiresAt time.Time) (Recipient, bool, error)
	// PendingPasswordHash returns the password hash of the account a usable
	// token belongs to, without using the token. Unknown, used and expired
	// tokens return ErrTokenInvalid.
	PendingPasswordHash(ctx context.Context, tokenHash []byte) (passwordHash string, err error)
	// ConsumeToken marks the token used and the email verified, and returns
	// the owner. Unknown, used and expired tokens all return ErrTokenInvalid.
	ConsumeToken(ctx context.Context, tokenHash []byte) (userID uuid.UUID, err error)
}
