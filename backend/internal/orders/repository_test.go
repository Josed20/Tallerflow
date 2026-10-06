package orders

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpdateStageRejectsCompletingPendingStage(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, stageID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 6, 20, 0, 0, 0, time.UTC)
	repository.now = func() time.Time { return now }

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at FROM order_stages WHERE order_id = $1 AND position = $2")).
		WithArgs(orderID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at"}).
			AddRow(stageID, workshopID, orderID, 1, "CORTE", "PENDING", nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE order_stages SET status = $1, completed_at = $2 WHERE id = $3 AND status = 'IN_PROGRESS'")).
		WithArgs("COMPLETED", now, stageID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := repository.UpdateStage(context.Background(), workshopID, orderID, 1, "COMPLETED")

	require.ErrorIs(t, err, ErrOrderInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

type orderRepositoryTenantRunner struct {
	db *gorm.DB
}

func (r orderRepositoryTenantRunner) WithinTenant(_ context.Context, _ uuid.UUID, fn database.TenantFunc) error {
	return fn(r.db)
}

func newOrderRepositoryFixture(t *testing.T) (*PostgresRepository, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB, PreferSimpleProtocol: true}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)

	repository, err := NewPostgresRepository(orderRepositoryTenantRunner{db: db})
	require.NoError(t, err)
	return repository, mock
}
