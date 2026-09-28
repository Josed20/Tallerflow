package passwordreset

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (userID uuid.UUID, exists bool, err error)
	CreateResetToken(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time, createdAt time.Time) error
	ConsumeResetTokenAndChangePassword(ctx context.Context, tokenHash []byte, consumedAt time.Time, hashPassword func() (string, error)) error
	// AllowRequest atomically reserves a recovery request slot for the IP and email.
	AllowRequest(ctx context.Context, ip, email string) (bool, error)
	// AllowConsume atomically reserves a recovery consumption slot for the IP and token digest.
	AllowConsume(ctx context.Context, ip string, tokenHash []byte) (bool, error)
}
