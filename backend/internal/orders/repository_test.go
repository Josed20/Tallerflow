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
	actorID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM orders WHERE id = $1 FOR UPDATE")).WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("ACTIVE"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = $1 AND position = $2 FOR UPDATE")).
		WithArgs(orderID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at", "completed_by"}).
			AddRow(stageID, workshopID, orderID, 1, "CORTE", "PENDING", nil, nil))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE order_stages SET status = $1, completed_at = $2, completed_by = $3 WHERE id = $4 AND status = 'IN_PROGRESS'")).
		WithArgs("COMPLETED", now, actorID, stageID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := repository.UpdateStage(context.Background(), workshopID, orderID, 1, actorID, "COMPLETED")

	require.ErrorIs(t, err, ErrOrderInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrderRejectsStaleVersionBeforeWriting(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, clientID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	promisedAt := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT client_id, product, quantity, promised_at, status, version FROM orders WHERE id = $1 FOR UPDATE")).WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "product", "quantity", "promised_at", "status", "version"}).AddRow(clientID, "Camisa", 2, promisedAt, "ACTIVE", 3))
	_, err := repository.Update(context.Background(), workshopID, orderID, actorID, UpdateOrderInput{
		ClientID: clientID, Product: "Camisa", Quantity: 2, PromisedAt: promisedAt, Version: 2,
	})
	require.ErrorIs(t, err, ErrOrderVersionConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrderRejectsClientOutsideWorkshop(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, clientID, otherClientID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	promisedAt := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT client_id, product, quantity, promised_at, status, version FROM orders WHERE id = $1 FOR UPDATE")).WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "product", "quantity", "promised_at", "status", "version"}).AddRow(clientID, "Camisa", 2, promisedAt, "ACTIVE", 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM clients WHERE id = $1")).WithArgs(otherClientID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	_, err := repository.Update(context.Background(), workshopID, orderID, actorID, UpdateOrderInput{
		ClientID: otherClientID, Product: "Camisa", Quantity: 2, PromisedAt: promisedAt, Version: 1,
	})
	require.ErrorIs(t, err, ErrOrderInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateOrderRejectsCompletedOrder(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, clientID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	promisedAt := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery(regexp.QuoteMeta("SELECT client_id, product, quantity, promised_at, status, version FROM orders WHERE id = $1 FOR UPDATE")).WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"client_id", "product", "quantity", "promised_at", "status", "version"}).AddRow(clientID, "Camisa", 2, promisedAt, "COMPLETED", 1))
	_, err := repository.Update(context.Background(), workshopID, orderID, actorID, UpdateOrderInput{
		ClientID: clientID, Product: "Camisa", Quantity: 3, PromisedAt: promisedAt, Version: 1,
	})
	require.ErrorIs(t, err, ErrOrderNotEditable)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReopenStageRejectsWhenNextStageHasEvidence(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, stageID, nextID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM orders WHERE id = $1 FOR UPDATE")).WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("ACTIVE"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = $1 AND position = $2 FOR UPDATE")).WithArgs(orderID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at", "completed_by"}).AddRow(stageID, workshopID, orderID, 1, "CORTE", "COMPLETED", time.Now(), uuid.New()))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = $1 AND position = $2 FOR UPDATE")).WithArgs(orderID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at", "completed_by"}).AddRow(nextID, workshopID, orderID, 2, "COSTURA", "IN_PROGRESS", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM stage_photos WHERE order_id = $1 AND stage_position = $2")).WithArgs(orderID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	_, err := repository.ReopenStage(context.Background(), workshopID, orderID, 1, uuid.New(), "Se avanzó por error")
	require.ErrorIs(t, err, ErrNextStageHasEvidence)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReopenStageRecordsCorrectionAndRestoresPreviousStage(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, stageID, nextID, actorID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 13, 16, 0, 0, 0, time.UTC)
	repository.now = func() time.Time { return now }
	mock.ExpectQuery(regexp.QuoteMeta("SELECT status FROM orders WHERE id = $1 FOR UPDATE")).WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("ACTIVE"))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = $1 AND position = $2 FOR UPDATE")).WithArgs(orderID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at", "completed_by"}).AddRow(stageID, workshopID, orderID, 1, "CORTE", "COMPLETED", now, actorID))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = $1 AND position = $2 FOR UPDATE")).WithArgs(orderID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at", "completed_by"}).AddRow(nextID, workshopID, orderID, 2, "COSTURA", "IN_PROGRESS", nil, nil))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM stage_photos WHERE order_id = $1 AND stage_position = $2")).WithArgs(orderID, 2).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectExec("UPDATE order_stages SET status = 'PENDING'").WithArgs(nextID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE order_stages SET status = 'IN_PROGRESS', completed_at = NULL, completed_by = NULL").WithArgs(stageID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE orders SET status = 'ACTIVE'").WithArgs(now, orderID).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec(regexp.QuoteMeta("jsonb_build_object('order_id', $3::text, 'stage_position', $4::integer, 'reason', $5::text)")).WithArgs(workshopID, actorID, orderID.String(), 1, "Se marcó por error").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE id = $1")).WithArgs(stageID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "workshop_id", "order_id", "position", "name", "status", "completed_at", "completed_by"}).AddRow(stageID, workshopID, orderID, 1, "CORTE", "IN_PROGRESS", nil, nil))

	stage, err := repository.ReopenStage(context.Background(), workshopID, orderID, 1, actorID, "Se marcó por error")
	require.NoError(t, err)
	require.Equal(t, "IN_PROGRESS", stage.Status)
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
