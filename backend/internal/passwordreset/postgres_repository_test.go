package passwordreset

import (
	"context"
	"crypto/sha256"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{
		DisableAutomaticPing: true,
		Logger:               logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	return db, mock, func() { _ = sqlDB.Close() }
}

func TestPostgresRepositoryFindUserByEmail(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewPostgresRepository(db, time.Now)
	require.NoError(t, err)

	ctx := context.Background()
	testUserID := uuid.New()

	t.Run("returns user id when email exists", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM users WHERE lower(email) = lower($1)`)).
			WithArgs("test@tallerflow.pe").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(testUserID))

		id, exists, err := repo.FindUserByEmail(ctx, "test@tallerflow.pe")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, testUserID, id)
	})

	t.Run("returns false when user does not exist", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id FROM users WHERE lower(email) = lower($1)`)).
			WithArgs("unknown@tallerflow.pe").
			WillReturnRows(sqlmock.NewRows([]string{"id"}))

		id, exists, err := repo.FindUserByEmail(ctx, "unknown@tallerflow.pe")
		require.NoError(t, err)
		require.False(t, exists)
		require.Equal(t, uuid.Nil, id)
	})
}

func TestPostgresRepositoryCreateResetToken(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	repo, err := NewPostgresRepository(db, time.Now)
	require.NoError(t, err)

	ctx := context.Background()
	testUserID := uuid.New()
	tokenHash := sha256.Sum256([]byte("sample-token"))
	now := time.Now().UTC()
	expires := now.Add(1 * time.Hour)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM password_reset_tokens WHERE user_id = $1 AND used_at IS NULL`)).
		WithArgs(testUserID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (gen_random_uuid(), $1, $2, $3, $4)`)).
		WithArgs(testUserID, tokenHash[:], expires, now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.CreateResetToken(ctx, testUserID, tokenHash[:], expires, now)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryAllowRequestAtomicallyRecordsAttempts(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	repo, err := NewPostgresRepository(db, func() time.Time { return now })
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock(hashtext($1))`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`SELECT pg_advisory_xact_lock(hashtext($1))`)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FILTER (WHERE email = $1) AS subject_attempts, COUNT(*) FILTER (WHERE ip_prefix = $2) AS ip_attempts FROM login_attempts WHERE succeeded = true AND email LIKE $3 AND attempted_at >= $4`)).
		WithArgs(sqlmock.AnyArg(), "127.0.0.1", "password-reset:request:%", now.Add(-15*time.Minute)).
		WillReturnRows(sqlmock.NewRows([]string{"subject_attempts", "ip_attempts"}).AddRow(0, 0))
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO login_attempts (email, ip_prefix, succeeded, attempted_at) VALUES ($1, $2, true, $3)`)).
		WithArgs(sqlmock.AnyArg(), "127.0.0.1", now).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	allowed, err := repo.AllowRequest(context.Background(), "127.0.0.1", "owner@tallerflow.pe")

	require.NoError(t, err)
	require.True(t, allowed)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryConsumeResetTokenAndChangePassword(t *testing.T) {
	db, mock, cleanup := setupTestDB(t)
	defer cleanup()

	now := time.Date(2026, 9, 27, 15, 0, 0, 0, time.UTC)
	repo, err := NewPostgresRepository(db, func() time.Time { return now })
	require.NoError(t, err)

	ctx := context.Background()
	testTokenID := uuid.New()
	testUserID := uuid.New()
	tokenHash := sha256.Sum256([]byte("sample-token"))
	expiresAt := now.Add(1 * time.Hour)

	t.Run("successfully consumes token, changes password and revokes sessions", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1 FOR UPDATE`)).
			WithArgs(tokenHash[:]).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "used_at"}).
				AddRow(testTokenID, testUserID, expiresAt, nil))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_credentials SET password_hash = $1, must_change_password = false, password_changed_at = $2, updated_at = $3 WHERE user_id = $4`)).
			WithArgs("new-argon-hash", now, now, testUserID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE password_reset_tokens SET used_at = $1 WHERE id = $2 AND used_at IS NULL`)).
			WithArgs(now, testTokenID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`)).
			WithArgs(now, testUserID).
			WillReturnResult(sqlmock.NewResult(0, 3)) // e.g. 3 sessions revoked
		mock.ExpectCommit()

		err := repo.ConsumeResetTokenAndChangePassword(ctx, tokenHash[:], now, func() (string, error) { return "new-argon-hash", nil })
		require.NoError(t, err)
		require.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("returns ErrTokenAlreadyUsed if used_at is not null", func(t *testing.T) {
		usedAt := now.Add(-10 * time.Minute)
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1 FOR UPDATE`)).
			WithArgs(tokenHash[:]).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "used_at"}).
				AddRow(testTokenID, testUserID, expiresAt, usedAt))
		mock.ExpectRollback()

		err := repo.ConsumeResetTokenAndChangePassword(ctx, tokenHash[:], now, func() (string, error) { return "new-argon-hash", nil })
		require.ErrorIs(t, err, ErrTokenAlreadyUsed)
	})

	t.Run("returns ErrTokenExpired if expires_at is before consumedAt", func(t *testing.T) {
		expiredAt := now.Add(-1 * time.Minute)
		mock.ExpectBegin()
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, user_id, expires_at, used_at FROM password_reset_tokens WHERE token_hash = $1 FOR UPDATE`)).
			WithArgs(tokenHash[:]).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "used_at"}).
				AddRow(testTokenID, testUserID, expiredAt, nil))
		mock.ExpectRollback()

		err := repo.ConsumeResetTokenAndChangePassword(ctx, tokenHash[:], now, func() (string, error) { return "new-argon-hash", nil })
		require.ErrorIs(t, err, ErrTokenExpired)
	})
}
