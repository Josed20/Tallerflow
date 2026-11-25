package console

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/analytics"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MetricDay struct {
	Day                time.Time  `json:"day"`
	FirstOpens         *int64     `json:"first_opens"`
	Registrations      *int64     `json:"registrations"`
	Logins             *int64     `json:"logins"`
	Activities         *int64     `json:"activities"`
	ActiveUsers        *int64     `json:"active_users"`
	ActiveWorkshops    *int64     `json:"active_workshops"`
	PlayDeviceInstalls *int64     `json:"play_device_installs"`
	PlayUserInstalls   *int64     `json:"play_user_installs"`
	PlayUninstalls     *int64     `json:"play_uninstalls"`
	PlayImportedAt     *time.Time `json:"play_imported_at"`
	CoverageStartedAt  time.Time  `json:"coverage_started_at"`
}

type PeriodTotals struct {
	ActiveUsers      *int64 `json:"active_users"`
	ActiveWorkshops  *int64 `json:"active_workshops"`
	CoverageComplete bool   `json:"coverage_complete"`
}

type MetricFilter struct {
	Start, End       time.Time
	Channel, Version string
	IncludeDemo      bool
}

func metricFilter(c *gin.Context, now time.Time) (MetricFilter, error) {
	end := now.UTC().Truncate(24 * time.Hour)
	f := MetricFilter{Start: end.AddDate(0, 0, -29), End: end, Channel: c.Query("channel"), Version: c.Query("version"), IncludeDemo: c.Query("include_demo") == "true"}
	var err error
	if raw := c.Query("start"); raw != "" {
		f.Start, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return f, err
		}
	}
	if raw := c.Query("end"); raw != "" {
		f.End, err = time.Parse("2006-01-02", raw)
		if err != nil {
			return f, err
		}
	}
	if f.Start.After(f.End) || f.End.After(end) || f.End.Sub(f.Start) > 365*24*time.Hour || f.Start.Before(end.AddDate(0, -25, 0)) || (f.Channel != "" && f.Channel != "web" && f.Channel != "android") || len(f.Version) > 64 || strings.ContainsAny(f.Version, "\r\n\x00") {
		return f, errors.New("invalid metrics filter")
	}
	return f, nil
}

func (h *Handler) Analytics(c *gin.Context) {
	f, err := metricFilter(c, h.service.now())
	if err != nil {
		httpx.RespondError(c, 422, "METRICS_FILTER_INVALID", "Review dates, channel and version.")
		return
	}
	rows := []MetricDay{}
	err = h.service.db.WithContext(c.Request.Context()).Raw("SELECT * FROM console_analytics(?::date,?::date,?,?,?)", f.Start.Format("2006-01-02"), f.End.Format("2006-01-02"), f.Channel, f.Version, f.IncludeDemo).Scan(&rows).Error
	if err != nil {
		httpx.RespondError(c, 500, "METRICS_UNAVAILABLE", "Metrics are unavailable.")
		return
	}
	var totals PeriodTotals
	err = h.service.db.WithContext(c.Request.Context()).Raw("SELECT * FROM console_analytics_totals(?::date,?::date,?,?,?)", f.Start.Format("2006-01-02"), f.End.Format("2006-01-02"), f.Channel, f.Version, f.IncludeDemo).Scan(&totals).Error
	if err != nil {
		httpx.RespondError(c, 500, "METRICS_UNAVAILABLE", "Metrics are unavailable.")
		return
	}
	c.JSON(200, gin.H{"data": rows, "totals": totals, "meta": gin.H{"timezone": "UTC", "telemetry_source": "client_reported", "play_source": "google_play_csv", "first_open_is_download": false, "anonymous_first_opens_include_unclassified_demos": true, "raw_retention_days": 90, "play_retention_months": 25}})
}

func (h *Handler) SetDemo(c *gin.Context) {
	if !h.requireMutation(c) {
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, 422, "WORKSHOP_INVALID", "Invalid workshop.")
		return
	}
	var in struct {
		IsDemo *bool `json:"is_demo"`
	}
	if c.ShouldBindJSON(&in) != nil || in.IsDemo == nil {
		httpx.RespondError(c, 422, "DEMO_INVALID", "is_demo must be a boolean.")
		return
	}
	if err := h.service.db.WithContext(c.Request.Context()).Exec("SELECT console_set_demo(?,?,?)", id, *in.IsDemo, consoleSession(c).AdminID).Error; err != nil {
		httpx.RespondError(c, 409, "DEMO_UPDATE_FAILED", "Unable to update demo classification.")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ImportPlay(c *gin.Context) {
	if !h.requireMutation(c) {
		return
	}
	// Read at most 4 MiB of CSV plus JSON escaping overhead. No remote URLs or credentials are accepted.
	var in struct {
		CSV        string `json:"csv"`
		SourceName string `json:"source_name"`
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, analytics.MaxCSVBytes*2+4096))
	decoder.DisallowUnknownFields()
	var extra any
	if decoder.Decode(&in) != nil || decoder.Decode(&extra) != io.EOF || len(in.CSV) > analytics.MaxCSVBytes {
		httpx.RespondError(c, 422, "PLAY_CSV_INVALID", "CSV is invalid or exceeds 4 MiB.")
		return
	}
	name := strings.TrimSpace(in.SourceName)
	if name == "" || len(name) > 128 || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\r\n\x00") {
		httpx.RespondError(c, 422, "PLAY_SOURCE_INVALID", "Use a report filename without a path.")
		return
	}
	rows, err := analytics.ParsePlayCSV([]byte(in.CSV), h.service.now())
	if err != nil {
		httpx.RespondError(c, 422, "PLAY_CSV_INVALID", err.Error())
		return
	}
	digest := sha256.Sum256([]byte(in.CSV))
	checksum := hex.EncodeToString(digest[:])
	adminID := consoleSession(c).AdminID
	err = h.service.importPlayRows(c.Request.Context(), rows, name, adminID, checksum)
	if err != nil {
		httpx.RespondError(c, 500, "PLAY_IMPORT_FAILED", "The report was not imported.")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"rows": len(rows), "source": "google_play_csv", "checksum": checksum}})
}

// importPlayRows preserves an existing metric when its CSV header is omitted.
// A present blank/N/A value explicitly clears it to NULL.
func (s *Service) importPlayRows(ctx context.Context, rows []analytics.PlayRow, name string, adminID uuid.UUID, checksum string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			err := tx.Exec(`INSERT INTO analytics_play_daily(report_date,package_name,dimension,version,device_installs,user_installs,uninstalls,source_name,imported_by,checksum)
    VALUES (?::date,?,?,?,?,?,?,?,?,?) ON CONFLICT(report_date,package_name,dimension,version) DO UPDATE SET
    device_installs=CASE WHEN ? THEN excluded.device_installs ELSE analytics_play_daily.device_installs END,
    user_installs=CASE WHEN ? THEN excluded.user_installs ELSE analytics_play_daily.user_installs END,
    uninstalls=CASE WHEN ? THEN excluded.uninstalls ELSE analytics_play_daily.uninstalls END,
    source_name=excluded.source_name,imported_by=excluded.imported_by,checksum=excluded.checksum,imported_at=now()
    WHERE analytics_play_daily.checksum IS DISTINCT FROM excluded.checksum`, row.Date.Format("2006-01-02"), analytics.PlayPackage, row.Dimension, row.Version, row.DeviceInstalls, row.UserInstalls, row.Uninstalls, name, adminID, checksum, row.HasDeviceInstalls, row.HasUserInstalls, row.HasUninstalls).Error
			if err != nil {
				return err
			}
		}
		return tx.Exec(`INSERT INTO console_audit_events(actor_admin_id,event_type,reason,details) VALUES (?,'PLAY_CSV_IMPORTED','Google Play installs report',jsonb_build_object('source_name',?::text,'rows',?::integer,'sha256',?::text))`, adminID, name, len(rows), checksum).Error
	})
}
