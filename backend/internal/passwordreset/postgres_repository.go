package passwordreset

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db      *gorm.DB
	clock   func() time.Time
	limiter *rateLimiter
}

func NewPostgresRepository(db *gorm.DB, clock func() time.Time) (*PostgresRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database handle is required")
	}
	if clock == nil {
		clock = time.Now
	}
	return &PostgresRepository{
		db:      db,
		clock:   clock,
		limiter: newRateLimiter(clock, 15*time.Minute, 5),
	}, nil
}

func (r *PostgresRepository) FindUserByEmail(ctx context.Context, email string) (uuid.UUID, bool, error) {
	const query = `SELECT id FROM users WHERE lower(email) = lower(?)`
	var row struct {
		ID uuid.UUID
	}
	result := r.db.WithContext(ctx).Raw(query, strings.TrimSpace(email)).Scan(&row)
	if result.Error != nil {
		return uuid.Nil, false, fmt.Errorf("find user by email: %w", result.Error)
	}
	if result.RowsAffected == 0 || row.ID == uuid.Nil {
		return uuid.Nil, false, nil
	}
	return row.ID, true, nil
}

func (r *PostgresRepository) CreateResetToken(ctx context.Context, userID uuid.UUID, tokenHash []byte, expiresAt time.Time, createdAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM password_reset_tokens WHERE user_id = ? AND used_at IS NULL`, userID).Error; err != nil {
			return fmt.Errorf("delete old reset tokens: %w", err)
		}
		const insertQuery = `INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (gen_random_uuid(), ?, ?, ?, ?)`
		if err := tx.Exec(insertQuery, userID, tokenHash, expiresAt.UTC(), createdAt.UTC()).Error; err != nil {
			return fmt.Errorf("insert reset token: %w", err)
		}
		return nil
	})
}

func (r *PostgresRepository) ConsumeResetTokenAndChangePassword(ctx context.Context, tokenHash []byte, newPasswordHash string, consumedAt time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tokenRow struct {
			ID        uuid.UUID
			UserID    uuid.UUID
			ExpiresAt time.Time
			UsedAt    *time.Time
		}
		const findQuery = `SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = ? FOR UPDATE`
		result := tx.Raw(findQuery, tokenHash).Scan(&tokenRow)
		if result.Error != nil {
			return fmt.Errorf("lock reset token: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return ErrTokenInvalid
		}
		if tokenRow.UsedAt != nil {
			return ErrTokenAlreadyUsed
		}
		if !tokenRow.ExpiresAt.After(consumedAt) {
			return ErrTokenExpired
		}

		if err := tx.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE id = ? AND used_at IS NULL`, consumedAt.UTC(), tokenRow.ID).Error; err != nil {
			return fmt.Errorf("mark token used: %w", err)
		}

		updateRes := tx.Exec(`UPDATE user_credentials SET password_hash = ?, must_change_password = false, password_changed_at = ?, updated_at = ? WHERE user_id = ?`,
			newPasswordHash, consumedAt.UTC(), consumedAt.UTC(), tokenRow.UserID)
		if updateRes.Error != nil {
			return fmt.Errorf("update user credential: %w", updateRes.Error)
		}
		if updateRes.RowsAffected != 1 {
			return fmt.Errorf("update user credential: expected 1 row, got %d", updateRes.RowsAffected)
		}

		if err := tx.Exec(`UPDATE user_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, consumedAt.UTC(), tokenRow.UserID).Error; err != nil {
			return fmt.Errorf("revoke user sessions: %w", err)
		}

		return nil
	})
}

func (r *PostgresRepository) AllowRequest(_ context.Context, ip, email string) (bool, error) {
	return r.limiter.allow(ip, email), nil
}

func (r *PostgresRepository) RecordRequestFailure(_ context.Context, ip, email string) error {
	r.limiter.record(ip, email)
	return nil
}

type rateLimiter struct {
	mu          sync.Mutex
	clock       func() time.Time
	window      time.Duration
	maxAttempts int
	records     map[string][]time.Time
}

func newRateLimiter(clock func() time.Time, window time.Duration, maxAttempts int) *rateLimiter {
	return &rateLimiter{
		clock:       clock,
		window:      window,
		maxAttempts: maxAttempts,
		records:     make(map[string][]time.Time),
	}
}

func (l *rateLimiter) allow(ip, email string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	cutoff := now.Add(-l.window)

	for _, key := range []string{strings.TrimSpace(ip), strings.ToLower(strings.TrimSpace(email))} {
		if key == "" {
			continue
		}
		timestamps := l.records[key]
		valid := timestamps[:0]
		for _, t := range timestamps {
			if t.After(cutoff) {
				valid = append(valid, t)
			}
		}
		l.records[key] = valid
		if len(valid) >= l.maxAttempts {
			return false
		}
	}
	return true
}

func (l *rateLimiter) record(ip, email string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.clock()
	for _, key := range []string{strings.TrimSpace(ip), strings.ToLower(strings.TrimSpace(email))} {
		if key != "" {
			l.records[key] = append(l.records[key], now)
		}
	}
}
