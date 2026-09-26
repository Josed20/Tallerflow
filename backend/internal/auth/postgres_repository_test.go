package auth

import (
	"context"
	"errors"
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

const credentialSelect = `SELECT u.id AS user_id, c.password_hash, (u.status = 'ACTIVE') AS active, c.must_change_password FROM users AS u JOIN user_credentials AS c ON c.user_id = u.id WHERE u.email = $1 LIMIT 1`

func TestPostgresRepositoryNormalizesCredentialEmail(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(credentialSelect)).
		WithArgs("owner@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "password_hash", "active", "must_change_password"}).
			AddRow(userID, "argon-hash", true, true))

	credential, err := repository.FindByEmail(context.Background(), "  OWNER@Example.COM ")

	require.NoError(t, err)
	require.Equal(t, &Credential{UserID: userID, PasswordHash: "argon-hash", Active: true, MustChangePassword: true}, credential)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryReturnsInactiveCredential(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	userID := uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta(credentialSelect)).
		WithArgs("inactive@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "password_hash", "active", "must_change_password"}).
			AddRow(userID, "argon-hash", false, false))

	credential, err := repository.FindByEmail(context.Background(), "inactive@example.com")

	require.NoError(t, err)
	require.False(t, credential.Active)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryMapsMissingCredential(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	mock.ExpectQuery(regexp.QuoteMeta(credentialSelect)).
		WithArgs("missing@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "password_hash", "active", "must_change_password"}))

	credential, err := repository.FindByEmail(context.Background(), "missing@example.com")

	require.Nil(t, credential)
	require.ErrorIs(t, err, ErrCredentialNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryPreservesCredentialOperationalError(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	driverErr := errors.New("database offline")
	mock.ExpectQuery(regexp.QuoteMeta(credentialSelect)).
		WithArgs("owner@example.com").
		WillReturnError(driverErr)

	_, err := repository.FindByEmail(context.Background(), "owner@example.com")

	require.ErrorIs(t, err, driverErr)
	require.NotErrorIs(t, err, ErrCredentialNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryFindsOnlyActiveSession(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	sessionID, userID := uuid.New(), uuid.New()
	digest := [32]byte{1, 2, 3}
	expiresAt := testNow.Add(time.Hour)
	query := `SELECT id, user_id, expires_at, revoked_at FROM user_sessions WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2 LIMIT 1`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(digest[:], testNow).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "revoked_at"}).AddRow(sessionID, userID, expiresAt, nil))

	session, err := repository.FindActiveByTokenHash(context.Background(), digest, testNow)

	require.NoError(t, err)
	require.Equal(t, Session{ID: sessionID, UserID: userID, ExpiresAt: expiresAt}, session)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryMapsExpiredOrRevokedSession(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	digest := [32]byte{9}
	query := `SELECT id, user_id, expires_at, revoked_at FROM user_sessions WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > $2 LIMIT 1`
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(digest[:], testNow).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "expires_at", "revoked_at"}))

	_, err := repository.FindActiveByTokenHash(context.Background(), digest, testNow)

	require.ErrorIs(t, err, ErrSessionInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryRevokesSessions(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	userID := uuid.New()
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`)).
		WithArgs(testNow, userID).
		WillReturnResult(sqlmock.NewResult(0, 2))

	err := repository.RevokeAllForUser(context.Background(), userID, testNow)

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryResolvesExactlyOneActiveMembership(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	userID, workshopID := uuid.New(), uuid.New()
	query := `SELECT workshop_id, role FROM resolve_active_memberships($1)`
	mock.ExpectQuery(regexp.QuoteMeta(query)).WithArgs(userID).
		WillReturnRows(sqlmock.NewRows([]string{"workshop_id", "role"}).AddRow(workshopID, "OWNER"))

	membership, err := repository.ResolveActive(context.Background(), userID)

	require.NoError(t, err)
	require.Equal(t, ActiveMembership{WorkshopID: workshopID, Role: "OWNER"}, membership)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresRepositoryRejectsZeroOrMultipleActiveMemberships(t *testing.T) {
	for _, test := range []struct {
		name string
		rows *sqlmock.Rows
	}{
		{name: "zero", rows: sqlmock.NewRows([]string{"workshop_id", "role"})},
		{name: "multiple", rows: sqlmock.NewRows([]string{"workshop_id", "role"}).AddRow(uuid.New(), "OWNER").AddRow(uuid.New(), "ADMIN")},
	} {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newPostgresRepositoryFixture(t)
			userID := uuid.New()
			mock.ExpectQuery(regexp.QuoteMeta(`SELECT workshop_id, role FROM resolve_active_memberships($1)`)).WithArgs(userID).WillReturnRows(test.rows)

			_, err := repository.ResolveActive(context.Background(), userID)

			require.ErrorIs(t, err, ErrInvalidCredentials)
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLoginSessionRollbackPreservesPreviousSessions(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	insertFailure := errors.New("insert failed")
	prepared := preparedSessionFixture()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT password_hash FROM user_credentials WHERE user_id = $1 FOR UPDATE`)).
		WithArgs(prepared.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow("verified-hash"))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`)).
		WithArgs(prepared.CreatedAt, prepared.UserID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("INSERT INTO user_sessions").WillReturnError(insertFailure)
	mock.ExpectRollback()

	_, err := repository.InsertForCredential(context.Background(), "verified-hash", prepared)

	require.ErrorIs(t, err, insertFailure)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPasswordRotationRollbackPreservesOldCredential(t *testing.T) {
	repository, mock := newPostgresRepositoryFixture(t)
	insertFailure := errors.New("insert failed")
	prepared := preparedSessionFixture()
	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT password_hash FROM user_credentials WHERE user_id = $1 FOR UPDATE`)).
		WithArgs(prepared.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"password_hash"}).AddRow("old-hash"))
	mock.ExpectExec("UPDATE user_credentials SET password_hash").
		WithArgs("new-hash", prepared.CreatedAt, prepared.CreatedAt, prepared.UserID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`)).
		WithArgs(prepared.CreatedAt, prepared.UserID).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("INSERT INTO user_sessions").WillReturnError(insertFailure)
	mock.ExpectRollback()

	_, err := repository.ChangePasswordAndInsert(context.Background(), prepared.UserID, "old-hash", "new-hash", prepared.CreatedAt, prepared)

	require.ErrorIs(t, err, insertFailure)
	require.NoError(t, mock.ExpectationsWereMet())
}

func preparedSessionFixture() NewSession {
	return NewSession{
		UserID:        testUserID,
		TokenHash:     [32]byte{1},
		CSRFTokenHash: [32]byte{2},
		CreatedAt:     testNow,
		ExpiresAt:     testNow.Add(8 * time.Hour),
	}
}

func newPostgresRepositoryFixture(t *testing.T) (*PostgresRepository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	repository, err := NewPostgresRepository(db, fixedClock(testNow))
	require.NoError(t, err)
	return repository, mock
}
