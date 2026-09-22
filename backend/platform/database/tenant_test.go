package database

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testRunner(t *testing.T) (*GormTenantRunner, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	runner, err := NewTenantRunner(db)
	require.NoError(t, err)
	return runner, mock
}

func TestTenantRunnerRejectsMissingTenant(t *testing.T) {
	runner, _ := testRunner(t)
	err := runner.WithinTenant(context.Background(), uuid.Nil, func(*gorm.DB) error { return nil })
	require.ErrorIs(t, err, ErrTenantRequired)
}

func TestTenantRunnerSetsContextAndCommits(t *testing.T) {
	runner, mock := testRunner(t)
	id := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT set_config('app.workshop_id', $1, true)")).WithArgs(id.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := runner.WithinTenant(context.Background(), id, func(*gorm.DB) error { return nil })
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTenantRunnerRollsBackOperationFailure(t *testing.T) {
	runner, mock := testRunner(t)
	id := uuid.New()
	want := errors.New("operation failed")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("SELECT set_config('app.workshop_id', $1, true)")).WithArgs(id.String()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectRollback()

	err := runner.WithinTenant(context.Background(), id, func(*gorm.DB) error { return want })
	require.ErrorIs(t, err, want)
	require.NoError(t, mock.ExpectationsWereMet())
}
