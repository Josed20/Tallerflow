package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID             uuid.UUID `json:"id"`
	InstallationID uuid.UUID `json:"installation_id"`
	Type           string    `json:"type"`
	OccurredAt     time.Time `json:"occurred_at"`
	Channel        string    `json:"channel"`
	Version        string    `json:"version"`
}

var versionPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]{0,63}$`)

func (e Event) Validate(now time.Time, anonymous bool) error {
	if e.ID == uuid.Nil || e.InstallationID == uuid.Nil || (e.Channel != "android" && e.Channel != "web") || !versionPattern.MatchString(e.Version) {
		return errors.New("invalid event metadata")
	}
	if e.OccurredAt.Before(now.Add(-7*24*time.Hour)) || e.OccurredAt.After(now.Add(5*time.Minute)) {
		return errors.New("invalid event time")
	}
	if anonymous {
		if e.Type != "first_open" {
			return errors.New("only first_open is anonymous")
		}
	} else if e.Type != "login" && e.Type != "registration" && e.Type != "activity" {
		return errors.New("invalid authenticated event")
	}
	return nil
}

var ErrEvidenceInvalid = errors.New("server evidence required")

type Service struct {
	DB      *gorm.DB
	Tenants database.TenantRunner
}

func (s *Service) Record(ctx context.Context, e Event, p *httpx.Principal) error {
	if err := e.Validate(time.Now().UTC(), p == nil); err != nil {
		return err
	}
	if p == nil {
		return s.DB.WithContext(ctx).Exec("SELECT analytics_first_open(?, ?, ?, ?, ?)", e.ID, e.InstallationID, e.OccurredAt, e.Channel, e.Version).Error
	}
	return s.Tenants.WithinTenant(ctx, p.WorkshopID, func(tx *gorm.DB) error {
		var accepted bool
		err := tx.Raw(`SELECT analytics_record_event(?,?,?,?,?,?,?,?)`, e.ID, e.InstallationID, e.Type, e.OccurredAt, e.Channel, e.Version, p.WorkshopID, p.UserID).Row().Scan(&accepted)
		if err != nil {
			return err
		}
		if !accepted {
			return ErrEvidenceInvalid
		}
		return nil
	})
}

// StartRetention never blocks API startup. Errors retry at the next daily run;
// callers cancel this worker as part of application shutdown.
func (s *Service) StartRetention() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			run, stop := context.WithTimeout(ctx, 30*time.Second)
			err := s.DB.WithContext(run).Exec("SELECT analytics_prune()").Error
			stop()
			if err != nil && ctx.Err() == nil {
				log.Printf("analytics retention failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return cancel
}

func RegisterRoutes(r gin.IRouter, s *Service, session gin.HandlerFunc, origins []string) {
	limiter := security.NewMemoryRateLimiter([]byte("analytics-events"), time.Now, time.Minute, 120, 10000)
	handle := func(anonymous bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			origin := c.GetHeader("Origin")
			allowed := origin == ""
			for _, v := range origins {
				if origin == v {
					allowed = true
				}
			}
			if !allowed {
				httpx.RespondError(c, 403, "ORIGIN_INVALID", "Request origin is not allowed.")
				return
			}
			if !limiter.Allow(strings.TrimSpace(c.ClientIP())) {
				c.Header("Retry-After", "60")
				httpx.RespondError(c, 429, "ANALYTICS_RATE_LIMITED", "Too many events.")
				return
			}
			var e Event
			decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 2048))
			decoder.DisallowUnknownFields()
			err := decoder.Decode(&e)
			var extra any
			if err != nil || decoder.Decode(&extra) != io.EOF || e.Validate(time.Now().UTC(), anonymous) != nil {
				httpx.RespondError(c, 422, "EVENT_INVALID", "Invalid analytics event.")
				return
			}
			var principal *httpx.Principal
			if !anonymous {
				p, err := httpx.PrincipalFromGin(c)
				if err != nil {
					httpx.RespondError(c, 401, "SESSION_INVALID", "Session required.")
					return
				}
				principal = &p
			}
			if err := s.Record(c.Request.Context(), e, principal); err != nil {
				if errors.Is(err, ErrEvidenceInvalid) {
					httpx.RespondError(c, 422, "EVENT_EVIDENCE_REQUIRED", "Server evidence is required.")
					return
				}
				httpx.RespondError(c, 503, "ANALYTICS_UNAVAILABLE", "Try again later.")
				return
			}
			c.Status(http.StatusNoContent)
		}
	}
	r.POST("/api/v1/analytics/first-open", handle(true))
	r.POST("/api/v1/analytics/events", session, handle(false))
}
