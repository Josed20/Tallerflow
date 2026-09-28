package passwordreset

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	FindUserByEmail(ctx context.Context, email string) (userID uuid.UUID, exists bool, err error)
	CreateResetToken(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time, createdAt time.Time) error
	ConsumeResetTokenAndChangePassword(ctx context.Context, tokenHash []byte, newPasswordHash string, consumedAt time.Time) error
	AllowRequest(ctx context.Context, ip, email string) (bool, error)
	RecordRequestFailure(ctx context.Context, ip, email string) error
}
