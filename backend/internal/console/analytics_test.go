package console

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Josed20/Tallerflow/backend/internal/analytics"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestSummaryMapsNumericColumnNames(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	require.NoError(t, err)
	mock.ExpectQuery(`SELECT \* FROM console_summary_filtered\(\$1\)`).WithArgs(false).
		WillReturnRows(sqlmock.NewRows([]string{"created_last_7_days", "created_last_30_days", "active_last_7_days", "unverified_owners_over_48h", "orders_last_30_days"}).AddRow(7, 30, 5, 2, 41))
	summary, err := (&Service{db: db}).Summary(context.Background())
	require.NoError(t, err)
	require.EqualValues(t, 7, summary.CreatedLast7Days)
	require.EqualValues(t, 30, summary.CreatedLast30Days)
	require.EqualValues(t, 5, summary.ActiveLast7Days)
	require.EqualValues(t, 2, summary.UnverifiedOwnersOver48h)
	require.EqualValues(t, 41, summary.OrdersLast30Days)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPartialPlayImportPreservesOmittedColumnsAndRollsBackAuditFailure(t *testing.T) {
	for _, failAudit := range []bool{false, true} {
		t.Run(map[bool]string{false: "commit", true: "audit failure rolls back"}[failAudit], func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer sqlDB.Close()
			db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
			require.NoError(t, err)
			now := time.Now().UTC()
			date := now.Format("2006-01-02")
			csv := "Date,Package Name,App Version Code,Daily Device Installs,Daily User Installs\n" + date + ",pe.tallerflow.app,03,8,\n"
			rows, err := analytics.ParsePlayCSV([]byte(csv), now)
			require.NoError(t, err)
			admin := uuid.New()
			mock.ExpectBegin()
			// Require conditional replacement for each metric. An omitted uninstalls
			// header must not turn a known persisted value into NULL; a blank user
			// installs header must clear it. Version 03 must address canonical row 3.
			query := `(?s)INSERT INTO analytics_play_daily.*device_installs=CASE WHEN \$11 THEN excluded.device_installs ELSE analytics_play_daily.device_installs END,.*user_installs=CASE WHEN \$12 THEN excluded.user_installs ELSE analytics_play_daily.user_installs END,.*uninstalls=CASE WHEN \$13 THEN excluded.uninstalls ELSE analytics_play_daily.uninstalls END,`
			mock.ExpectExec(query).WithArgs(date, analytics.PlayPackage, "app_version", "3", int64(8), nil, nil, "partial.csv", admin, "checksum", true, true, false).WillReturnResult(sqlmock.NewResult(0, 1))
			audit := mock.ExpectExec(`INSERT INTO console_audit_events`).WithArgs(admin, "partial.csv", 1, "checksum")
			if failAudit {
				audit.WillReturnError(errors.New("audit unavailable"))
				mock.ExpectRollback()
			} else {
				audit.WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectCommit()
			}
			err = (&Service{db: db}).importPlayRows(context.Background(), rows, "partial.csv", admin, "checksum")
			if failAudit {
				require.ErrorContains(t, err, "audit unavailable")
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestMetricsRejectUnboundedOrInvalidFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	for _, query := range []string{"start=2026-11-24&end=2026-11-23", "start=2024-11-23&end=2026-11-24", "end=2026-11-25", "channel=ios", "start=nope"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/?"+query, nil)
		_, err := metricFilter(c, now)
		require.Error(t, err, query)
	}
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/?channel=android&version=3&include_demo=true", nil)
	filter, err := metricFilter(c, now)
	require.NoError(t, err)
	require.True(t, filter.IncludeDemo)
	require.Equal(t, "3", filter.Version)
}
