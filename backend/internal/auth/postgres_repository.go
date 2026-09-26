package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PostgresRepository struct {
	db    *gorm.DB
	clock func() time.Time
}

func NewPostgresRepository(db *gorm.DB, clock func() time.Time) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("database is required")
	}
	if clock == nil {
		clock = time.Now
	}
	return &PostgresRepository{db: db, clock: clock}, nil
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*Credential, error) {
	const query = `SELECT u.id AS user_id, c.password_hash, (u.status = 'ACTIVE') AS active, c.must_change_password FROM users AS u JOIN user_credentials AS c ON c.user_id = u.id WHERE u.email = ? LIMIT 1`
	return r.findCredential(ctx, query, strings.ToLower(strings.TrimSpace(email)))
}

func (r *PostgresRepository) FindByUserID(ctx context.Context, userID uuid.UUID) (*Credential, error) {
	const query = `SELECT u.id AS user_id, c.password_hash, (u.status = 'ACTIVE') AS active, c.must_change_password FROM users AS u JOIN user_credentials AS c ON c.user_id = u.id WHERE u.id = ? LIMIT 1`
	return r.findCredential(ctx, query, userID)
}

func (r *PostgresRepository) findCredential(ctx context.Context, query string, argument any) (*Credential, error) {
	var credential Credential
	result := r.db.WithContext(ctx).Raw(query, argument).Scan(&credential)
	if result.Error != nil {
		return nil, fmt.Errorf("find credential: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return nil, ErrCredentialNotFound
	}
	return &credential, nil
}

func (r *PostgresRepository) Insert(ctx context.Context, created NewSession) (Session, error) {
	return insertSession(r.db.WithContext(ctx), created)
}

func (r *PostgresRepository) InsertForCredential(ctx context.Context, verifiedHash string, created NewSession) (Session, error) {
	var session Session
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		storedHash, err := lockCredentialHash(tx, created.UserID)
		if err != nil {
			return err
		}
		if storedHash != verifiedHash {
			return ErrCredentialChanged
		}
		if err := revokeActiveSessions(tx, created.UserID, created.CreatedAt); err != nil {
			return err
		}
		session, err = insertSession(tx, created)
		return err
	})
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func insertSession(db *gorm.DB, created NewSession) (Session, error) {
	const query = `INSERT INTO user_sessions (user_id, token_hash, csrf_token_hash, expires_at, created_at, last_seen_at, ip_prefix, user_agent) VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING id, user_id, expires_at, revoked_at`
	var session Session
	result := db.Raw(query,
		created.UserID, created.TokenHash[:], created.CSRFTokenHash[:], created.ExpiresAt,
		created.CreatedAt, created.CreatedAt, nullableString(created.Metadata.IPPrefix), nullableString(created.Metadata.UserAgent),
	).Scan(&session)
	if result.Error != nil {
		return Session{}, fmt.Errorf("insert session: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return Session{}, errors.New("insert session: no row returned")
	}
	return session, nil
}

func (r *PostgresRepository) FindActiveByTokenHash(ctx context.Context, digest [32]byte, now time.Time) (Session, error) {
	const query = `SELECT id, user_id, expires_at, revoked_at FROM user_sessions WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ? LIMIT 1`
	var session Session
	result := r.db.WithContext(ctx).Raw(query, digest[:], now).Scan(&session)
	if result.Error != nil {
		return Session{}, fmt.Errorf("find active session: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return Session{}, ErrSessionInvalid
	}
	return session, nil
}

func (r *PostgresRepository) Revoke(ctx context.Context, sessionID uuid.UUID, revokedAt time.Time) error {
	const query = `UPDATE user_sessions SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`
	if err := r.db.WithContext(ctx).Exec(query, revokedAt, sessionID).Error; err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	const query = `UPDATE user_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`
	if err := r.db.WithContext(ctx).Exec(query, revokedAt, userID).Error; err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ChangePasswordAndInsert(ctx context.Context, userID uuid.UUID, expectedHash, replacementHash string, changedAt time.Time, created NewSession) (Session, error) {
	var session Session
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		storedHash, err := lockCredentialHash(tx, userID)
		if err != nil {
			return err
		}
		if storedHash != expectedHash {
			return ErrCredentialChanged
		}
		result := tx.Exec(`UPDATE user_credentials SET password_hash = ?, must_change_password = false, password_changed_at = ?, updated_at = ? WHERE user_id = ?`, replacementHash, changedAt, changedAt, userID)
		if result.Error != nil {
			return fmt.Errorf("update credential: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return ErrCredentialNotFound
		}
		if err := revokeActiveSessions(tx, userID, changedAt); err != nil {
			return err
		}
		session, err = insertSession(tx, created)
		return err
	})
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func lockCredentialHash(tx *gorm.DB, userID uuid.UUID) (string, error) {
	var passwordHash string
	result := tx.Raw(`SELECT password_hash FROM user_credentials WHERE user_id = ? FOR UPDATE`, userID).Scan(&passwordHash)
	if result.Error != nil {
		return "", fmt.Errorf("lock credential: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return "", ErrCredentialChanged
	}
	return passwordHash, nil
}

func revokeActiveSessions(tx *gorm.DB, userID uuid.UUID, revokedAt time.Time) error {
	if err := tx.Exec(`UPDATE user_sessions SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, revokedAt, userID).Error; err != nil {
		return fmt.Errorf("revoke user sessions: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ResolveActive(ctx context.Context, userID uuid.UUID) (ActiveMembership, error) {
	const query = `SELECT workshop_id, role FROM resolve_active_memberships(?)`
	var memberships []ActiveMembership
	result := r.db.WithContext(ctx).Raw(query, userID).Scan(&memberships)
	if result.Error != nil {
		return ActiveMembership{}, fmt.Errorf("resolve active membership: %w", result.Error)
	}
	if len(memberships) != 1 {
		return ActiveMembership{}, ErrInvalidCredentials
	}
	return memberships[0], nil
}

func (r *PostgresRepository) Allow(ctx context.Context, email, ip string) (bool, error) {
	const query = `SELECT COUNT(*) FILTER (WHERE email = ?) AS email_ip_failures, COUNT(*) AS ip_failures FROM login_attempts WHERE succeeded = false AND ip_prefix = ? AND attempted_at >= ?`
	var failures struct {
		EmailIPFailures int64
		IPFailures      int64
	}
	windowStart := r.clock().UTC().Add(-15 * time.Minute)
	result := r.db.WithContext(ctx).Raw(query, strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(ip), windowStart).Scan(&failures)
	if result.Error != nil {
		return false, fmt.Errorf("check login limit: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return false, errors.New("check login limit: no row returned")
	}
	return failures.IPFailures < 5 && failures.EmailIPFailures < 5, nil
}

func (r *PostgresRepository) RecordFailure(ctx context.Context, email, ip string) error {
	const query = `INSERT INTO login_attempts (email, ip_prefix, succeeded, attempted_at) VALUES (?, ?, false, ?)`
	result := r.db.WithContext(ctx).Exec(query, strings.ToLower(strings.TrimSpace(email)), strings.TrimSpace(ip), r.clock().UTC())
	if result.Error != nil {
		return fmt.Errorf("record login failure: %w", result.Error)
	}
	if result.RowsAffected != 1 {
		return errors.New("record login failure: no row inserted")
	}
	return nil
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
