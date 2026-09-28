package passwordreset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db    *gorm.DB
	clock func() time.Time
}

const (
	recoveryRateWindow      = 15 * time.Minute
	recoveryRateMaxAttempts = 5
	recoveryRequestScope    = "request"
	recoveryConsumeScope    = "consume"
)

func NewPostgresRepository(db *gorm.DB, clock func() time.Time) (*PostgresRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("database handle is required")
	}
	if clock == nil {
		clock = time.Now
	}
	return &PostgresRepository{
		db:    db,
		clock: clock,
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

func (r *PostgresRepository) ConsumeResetTokenAndChangePassword(ctx context.Context, tokenHash []byte, consumedAt time.Time, hashPassword func() (string, error)) error {
	if hashPassword == nil {
		return fmt.Errorf("password hash callback is required")
	}
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

		newPasswordHash, err := hashPassword()
		if err != nil {
			return fmt.Errorf("hash new password: %w", err)
		}

		updateRes := tx.Exec(`UPDATE user_credentials SET password_hash = ?, must_change_password = false, password_changed_at = ?, updated_at = ? WHERE user_id = ?`,
			newPasswordHash, consumedAt.UTC(), consumedAt.UTC(), tokenRow.UserID)
		if updateRes.Error != nil {
			return fmt.Errorf("update user credential: %w", updateRes.Error)
		}
		if updateRes.RowsAffected != 1 {
			return fmt.Errorf("update user credential: expected 1 row, got %d", updateRes.RowsAffected)
		}

		if err := tx.Exec(`UPDATE password_reset_tokens SET used_at = ? WHERE id = ? AND used_at IS NULL`, consumedAt.UTC(), tokenRow.ID).Error; err != nil {
			return fmt.Errorf("mark token used: %w", err)
		}

		if err := tx.Exec(`UPDATE user_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, consumedAt.UTC(), tokenRow.UserID).Error; err != nil {
			return fmt.Errorf("revoke user sessions: %w", err)
		}

		return nil
	})
}

func (r *PostgresRepository) AllowRequest(ctx context.Context, ip, email string) (bool, error) {
	return r.reserveRecoveryAttempt(ctx, recoveryRequestScope, email, ip)
}

func (r *PostgresRepository) AllowConsume(ctx context.Context, ip string, tokenHash []byte) (bool, error) {
	return r.reserveRecoveryAttempt(ctx, recoveryConsumeScope, hex.EncodeToString(tokenHash), ip)
}

func (r *PostgresRepository) reserveRecoveryAttempt(ctx context.Context, scope, subject, ip string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	subjectKey := recoveryRateKey(scope, subject)
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown"
	}
	locks := []string{"password-reset:" + scope + ":ip:" + ip, "password-reset:" + scope + ":subject:" + subjectKey}
	sort.Strings(locks)
	now := r.clock().UTC()
	windowStart := now.Add(-recoveryRateWindow)
	scopePrefix := "password-reset:" + scope + ":%"

	allowed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, lockKey := range locks {
			if err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext(?))`, lockKey).Error; err != nil {
				return fmt.Errorf("lock recovery rate key: %w", err)
			}
		}

		var attempts struct {
			SubjectAttempts int64
			IPAttempts      int64
		}
		const countQuery = `SELECT COUNT(*) FILTER (WHERE email = ?) AS subject_attempts, COUNT(*) FILTER (WHERE ip_prefix = ?) AS ip_attempts FROM login_attempts WHERE succeeded = true AND email LIKE ? AND attempted_at >= ?`
		result := tx.Raw(countQuery, subjectKey, ip, scopePrefix, windowStart).Scan(&attempts)
		if result.Error != nil {
			return fmt.Errorf("count recovery attempts: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("count recovery attempts: no row returned")
		}
		if attempts.SubjectAttempts >= recoveryRateMaxAttempts || attempts.IPAttempts >= recoveryRateMaxAttempts {
			return nil
		}

		result = tx.Exec(`INSERT INTO login_attempts (email, ip_prefix, succeeded, attempted_at) VALUES (?, ?, true, ?)`, subjectKey, ip, now)
		if result.Error != nil {
			return fmt.Errorf("record recovery attempt: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("record recovery attempt: no row inserted")
		}
		allowed = true
		return nil
	})
	return allowed, err
}

func recoveryRateKey(scope, subject string) string {
	normalized := strings.ToLower(strings.TrimSpace(subject))
	digest := sha256.Sum256([]byte(scope + ":" + normalized))
	return "password-reset:" + scope + ":" + hex.EncodeToString(digest[:])
}
