package orders

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProductionSettingsSupportsFewerAndMoreThanSevenStages(t *testing.T) {
	for _, size := range []int{1, 6, 8, 20} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			input := ProductionSettings{}
			for i := 0; i < size; i++ {
				input.Stages = append(input.Stages, ProductionStage{fmt.Sprintf(" Etapa %d ", i), " Descripción para el cliente "})
			}
			require.NoError(t, validateProductionSettings(&input))
			require.Equal(t, "Etapa 0", input.Stages[0].Name)
		})
	}
	for _, stages := range [][]ProductionStage{nil, make([]ProductionStage, 21), {{"Corte", "Descripción"}, {" corte ", "Descripción"}}, {{"Corte", ""}}} {
		input := ProductionSettings{Stages: stages}
		require.ErrorIs(t, validateProductionSettings(&input), ErrOrderInvalid)
	}
	require.Len(t, defaultProductionSettings().Stages, 7)
}

func TestProductionTemplateSaveDoesNotRewriteExistingSnapshots(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, actorID := uuid.New(), uuid.New()
	input := ProductionSettings{Version: 3, Stages: []ProductionStage{{"Muestra", "Preparamos tu muestra."}, {"Entrega", "Listo para entregar."}}}
	// No UPDATE/DELETE of order_stages is allowed by these expectations.
	mock.ExpectExec("INSERT INTO production_settings").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE production_settings SET stages").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("INSERT INTO audit_events").WillReturnResult(sqlmock.NewResult(0, 1))
	settings, err := repository.SaveProductionSettings(context.Background(), workshopID, actorID, input)
	require.NoError(t, err)
	require.EqualValues(t, 4, settings.Version)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTemplateDefaultsOnlyWhenWorkshopHasNoConfiguration(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID := uuid.New()
	for _, custom := range []bool{false, true} {
		rows := sqlmock.NewRows([]string{"stages", "version"})
		if custom {
			rows.AddRow(`[{"name":"Muestra","client_description":"Tu muestra."}]`, 2)
		}
		mock.ExpectQuery("SELECT stages::text AS stages, version FROM production_settings").WithArgs(workshopID).WillReturnRows(rows)
		var settings *ProductionSettings
		err := repository.tenant.WithinTenant(context.Background(), workshopID, func(tx *gorm.DB) error {
			var err error
			settings, err = loadProductionSettings(tx, workshopID)
			return err
		})
		require.NoError(t, err)
		if custom {
			require.Len(t, settings.Stages, 1)
			require.Equal(t, "Muestra", settings.Stages[0].Name)
		} else {
			require.Len(t, settings.Stages, 7)
		}
	}
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestAssignmentRejectsUserOutsideWorkshop(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, actorID, foreignID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery("SELECT id, status, version, responsible_user_id, block_reason FROM orders").WithArgs(orderID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "status", "version", "block_reason"}).AddRow(orderID, "ACTIVE", 1, ""))
	mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM workshop_members WHERE workshop_id = $1 AND user_id = $2 AND status = 'ACTIVE'")).WithArgs(workshopID, foreignID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	_, err := repository.UpdateOperation(context.Background(), workshopID, orderID, actorID, OperationInput{ResponsibleUserID: &foreignID, Version: 1})
	require.ErrorIs(t, err, ErrOrderInvalid)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTwentyStageSnapshotCompletesAtItsOwnLastStage(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	workshopID, orderID, actorID, stageID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	mock.ExpectQuery("SELECT status FROM orders").WithArgs(orderID).WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("ACTIVE"))
	mock.ExpectQuery("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages").WithArgs(orderID, 20).
		WillReturnRows(sqlmock.NewRows([]string{"id", "position", "status"}).AddRow(stageID, 20, "IN_PROGRESS"))
	mock.ExpectExec("UPDATE order_stages SET status").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT max\\(position\\)").WithArgs(orderID).WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(20))
	mock.ExpectExec("UPDATE orders SET status = 'COMPLETED'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages").WithArgs(stageID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "position", "status"}).AddRow(stageID, 20, "COMPLETED"))
	stage, err := repository.UpdateStage(context.Background(), workshopID, orderID, 20, actorID, "COMPLETED")
	require.NoError(t, err)
	require.Equal(t, "COMPLETED", stage.Status)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestProductionSettingsRejectsOperatorsAndSupervisors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, role := range []string{"OPERATOR", "SUPERVISOR"} {
		handler := NewHandler(NewService(&handlerRepository{}))
		router := gin.New()
		router.PATCH("/api/v1/production-settings", principalMiddleware(role), handler.SaveProductionSettings)
		request := httptest.NewRequest(http.MethodPatch, "/api/v1/production-settings", strings.NewReader(`{"stages":[{"name":"Entrega","client_description":"Lista"}],"version":0}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		require.Equal(t, http.StatusForbidden, response.Code)
	}
}
func TestProductionTemplateRejectsStaleVersion(t *testing.T) {
	repository, mock := newOrderRepositoryFixture(t)
	input := defaultProductionSettings()
	input.Version = 2
	mock.ExpectExec("INSERT INTO production_settings").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("UPDATE production_settings SET stages").WillReturnResult(sqlmock.NewResult(0, 0))
	_, err := repository.SaveProductionSettings(context.Background(), uuid.New(), uuid.New(), input)
	require.ErrorIs(t, err, ErrOrderVersionConflict)
	require.NoError(t, mock.ExpectationsWereMet())
}
