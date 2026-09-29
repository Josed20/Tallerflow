package database

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestPingUsesUnderlyingSQLConnection(t *testing.T) {
	db, mock, cleanup := lifecycleDatabase(t)
	defer cleanup()
	mock.ExpectPing()

	require.NoError(t, Ping(context.Background(), db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPingWrapsUnderlyingFailure(t *testing.T) {
	db, mock, cleanup := lifecycleDatabase(t)
	defer cleanup()
	mock.ExpectPing().WillReturnError(errors.New("driver unavailable"))

	err := Ping(context.Background(), db)

	require.ErrorContains(t, err, "ping database")
	require.NotContains(t, err.Error(), "postgres://")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCloseUsesUnderlyingSQLConnection(t *testing.T) {
	db, mock, cleanup := lifecycleDatabase(t)
	defer cleanup()
	mock.ExpectClose()

	require.NoError(t, Close(db))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestRuntimeLoggerSuppressesSensitiveQueryParameters(t *testing.T) {
	filter, ok := runtimeGORMConfig().Logger.(gorm.ParamsFilter)
	require.True(t, ok, "the runtime logger must support GORM parameter filtering")

	query, params := filter.ParamsFilter(context.Background(), "UPDATE user_credentials SET password_hash = ?", "secret-argon-hash")

	require.Equal(t, "UPDATE user_credentials SET password_hash = ?", query)
	require.Empty(t, params, "password hashes and reset-token digests must never be interpolated into logs")
}

func lifecycleDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	return db, mock, func() { _ = sqlDB.Close() }
}
