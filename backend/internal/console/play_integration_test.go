package console

import (
	"context"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/analytics"
	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// Uses an existing migrated database, without starting containers or retaining fixtures.
func TestPostgresIntegrationPartialPlayReportsKeepOmittedMetricsAndClearExplicitBlanks(t *testing.T) {
	dsn := os.Getenv("TF_TEST_ADMIN_DATABASE_URL")
	if dsn == "" {
		t.Skip("TF_TEST_ADMIN_DATABASE_URL is not configured")
	}
	db, err := platformdb.Open(dsn)
	require.NoError(t, err)
	defer platformdb.Close(db)
	rollback := errors.New("rollback Play import fixtures")
	err = db.Transaction(func(tx *gorm.DB) error {
		admin := uuid.New()
		require.NoError(t, tx.Exec(`INSERT INTO console_admins(id,email,password_hash,totp_secret_ciphertext) VALUES (?,?,'fixture',decode('00','hex'))`, admin, admin.String()+"@example.test").Error)
		// A random canonical version isolates the fixtures without deleting existing reports.
		now := time.Now().UTC()
		date := now.Format("2006-01-02")
		version := strconv.FormatUint(uint64(uuid.New().ID()%1000000000+1000000000), 10)
		require.NoError(t, tx.Exec("SET LOCAL ROLE tallerflow_console").Error)
		service := &Service{db: tx}
		importCSV := func(headers, values, checksum string) {
			rows, err := analytics.ParsePlayCSV([]byte("Date,Package Name,App Version Code,"+headers+"\n"+date+",pe.tallerflow.app,0"+version+","+values+"\n"), now)
			require.NoError(t, err)
			require.NoError(t, service.importPlayRows(context.Background(), rows, "fixture.csv", admin, checksum))
		}
		type values struct {
			DeviceInstalls *int64
			UserInstalls   *int64
			Uninstalls     *int64
		}
		read := func() values {
			var row values
			require.NoError(t, tx.Raw("SELECT device_installs,user_installs,uninstalls FROM analytics_play_daily WHERE report_date=?::date AND version=?", date, version).Scan(&row).Error)
			return row
		}
		importCSV("Daily Device Installs,Daily User Installs,Daily Device Uninstalls", "10,5,2", "full")
		importCSV("Daily Device Installs", "8", "partial")
		row := read()
		require.EqualValues(t, 8, *row.DeviceInstalls)
		require.EqualValues(t, 5, *row.UserInstalls)
		require.EqualValues(t, 2, *row.Uninstalls)
		importCSV("Daily User Installs", "", "explicit-blank")
		row = read()
		require.EqualValues(t, 8, *row.DeviceInstalls)
		require.Nil(t, row.UserInstalls)
		require.EqualValues(t, 2, *row.Uninstalls)
		var count int64
		require.NoError(t, tx.Raw("SELECT count(*) FROM analytics_play_daily WHERE report_date=?::date AND version IN (?,?)", date, version, "0"+version).Scan(&count).Error)
		require.EqualValues(t, 1, count)
		return rollback
	})
	require.ErrorIs(t, err, rollback)
}
