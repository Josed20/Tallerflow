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

func lifecycleDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true})
	require.NoError(t, err)
	return db, mock, func() { _ = sqlDB.Close() }
}
