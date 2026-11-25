package orders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const trackingTokenBytes = 32

var (
	ErrTrackingNotFound            = errors.New("tracking link not found")
	ErrTrackingRecoveryUnavailable = errors.New("tracking recovery secret changed")
	trackingTokenFormat            = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
)

// TrackingOrder is the customer-facing view of an order: only what the
// workshop's customer may see, without internal ids, clients or notes.
type TrackingOrder struct {
	Specification string          `json:"specification"`
	WorkshopName  string          `json:"workshop_name"`
	Code          string          `json:"code"`
	Product       string          `json:"product"`
	Quantity      float64         `json:"quantity"`
	Unit          string          `json:"unit"`
	Status        string          `json:"status"`
	TrafficLight  TrafficLight    `json:"traffic_light"`
	PromisedAt    time.Time       `json:"promised_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
	Stages        []TrackingStage `json:"stages" gorm:"-"`
	Photos        []TrackingPhoto `json:"photos" gorm:"-"`
}

type TrackingStage struct {
	ClientDescription string     `json:"client_description"`
	Position          int        `json:"position"`
	Name              string     `json:"name"`
	Status            string     `json:"status"`
	CompletedAt       *time.Time `json:"completed_at"`
}

type TrackingPhoto struct {
	StagePosition int       `json:"stage_position"`
	PhotoData     string    `json:"photo_data"`
	ObjectKey     string    `json:"-"`
	Caption       string    `json:"caption"`
	CreatedAt     time.Time `json:"created_at"`
}

type TrackingRepository interface {
	ManageLink(ctx context.Context, workshopID, orderID, actorID uuid.UUID, action string) (string, error)
	FindByToken(ctx context.Context, tokenHash []byte) (*TrackingOrder, error)
}

type PostgresTrackingRepository struct {
	db     *gorm.DB
	tenant database.TenantRunner
	now    func() time.Time
	secret []byte
}

func NewPostgresTrackingRepository(db *gorm.DB, tenant database.TenantRunner, sessionPepper string) (*PostgresTrackingRepository, error) {
	if db == nil || tenant == nil {
		return nil, errors.New("database and tenant runner are required")
	}
	if strings.TrimSpace(sessionPepper) == "" {
		return nil, errors.New("tracking server secret is required")
	}
	return &PostgresTrackingRepository{db: db, tenant: tenant, now: time.Now, secret: []byte(sessionPepper)}, nil
}

// FindByToken resolves a public token to its workshop, then reads the order
// under that workshop's tenant context in the same transaction.
func (r *PostgresTrackingRepository) FindByToken(ctx context.Context, tokenHash []byte) (*TrackingOrder, error) {
	now := r.now().UTC()
	var order TrackingOrder
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT set_config('app.tracking_token_hash', ?, true)", hex.EncodeToString(tokenHash)).Error; err != nil {
			return fmt.Errorf("set tracking token context: %w", err)
		}
		var link struct {
			WorkshopID uuid.UUID
			OrderID    uuid.UUID
		}
		found := tx.Raw("SELECT workshop_id, order_id FROM order_tracking_links WHERE token_hash = ? AND revoked_at IS NULL", tokenHash).Scan(&link)
		if found.Error != nil {
			return found.Error
		}
		if found.RowsAffected == 0 {
			return ErrTrackingNotFound
		}
		if err := tx.Exec("SELECT set_config('app.workshop_id', ?, true)", link.WorkshopID.String()).Error; err != nil {
			return fmt.Errorf("set tenant context: %w", err)
		}

		row := tx.Raw(`
			SELECT w.name AS workshop_name, o.code, o.product, o.specification, o.quantity, o.unit, o.status, o.promised_at, o.updated_at,
				CASE
					WHEN o.status = 'COMPLETED' THEN 'COMPLETED'
					WHEN o.promised_at < ? THEN 'LATE'
					WHEN o.promised_at < ? THEN 'AT_RISK'
					ELSE 'ON_TIME'
				END AS traffic_light
			FROM orders o
			JOIN workshops w ON w.id = o.workshop_id
			WHERE o.id = ? AND w.status = 'ACTIVE'
 AND (o.tracking_completed_at IS NULL OR o.tracking_completed_at + interval '30 days' > ?)`, now, now.Add(48*time.Hour), link.OrderID, now).Scan(&order)
		if row.Error != nil {
			return row.Error
		}
		if row.RowsAffected == 0 {
			return ErrTrackingNotFound
		}
		if err := tx.Raw("SELECT position, name, client_description, status, completed_at FROM order_stages WHERE order_id = ? ORDER BY position", link.OrderID).Scan(&order.Stages).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT stage_position, COALESCE(photo_data, '') AS photo_data, COALESCE(object_key, '') AS object_key, caption, created_at FROM stage_photos WHERE order_id = ? ORDER BY created_at", link.OrderID).Scan(&order.Photos).Error
	})
	if err != nil {
		return nil, err
	}
	if order.Stages == nil {
		order.Stages = []TrackingStage{}
	}
	if order.Photos == nil {
		order.Photos = []TrackingPhoto{}
	}
	return &order, nil
}

type TrackingService struct {
	repo   TrackingRepository
	photos PhotoStore
}

func NewTrackingService(repo TrackingRepository, _ io.Reader, photos ...PhotoStore) *TrackingService {
	service := &TrackingService{repo: repo}
	if len(photos) > 0 {
		service.photos = photos[0]
	}
	return service
}

// CreateLink recovers or creates the stable link. Copying never rotates it.
func (s *TrackingService) CreateLink(ctx context.Context, workshopID, orderID, actorID uuid.UUID) (string, error) {
	return s.repo.ManageLink(ctx, workshopID, orderID, actorID, "recover")
}

func (s *TrackingService) Find(ctx context.Context, token string) (*TrackingOrder, error) {
	if !trackingTokenFormat.MatchString(token) {
		return nil, ErrTrackingNotFound
	}
	order, err := s.repo.FindByToken(ctx, trackingDigest(token))
	if err != nil {
		return nil, err
	}
	for i := range order.Photos {
		if err := resolvePhoto(ctx, s.photos, &order.Photos[i].PhotoData, order.Photos[i].ObjectKey); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func trackingDigest(token string) []byte {
	digest := sha256.Sum256([]byte(token))
	return digest[:]
}

type TrackingHandler struct {
	service *TrackingService
}

func NewTrackingHandler(service *TrackingService) *TrackingHandler {
	return &TrackingHandler{service: service}
}

func (h *TrackingHandler) CreateLink(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "Only workshop managers can share an order with its customer.")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}
	var token string
	action := c.Param("action")
	if action != "" && action != "replace" && action != "revoke" {
		httpx.RespondError(c, http.StatusBadRequest, "TRACKING_ACTION_INVALID", "Choose replace or revoke.")
		return
	}
	if action == "" {
		token, err = h.service.CreateLink(c.Request.Context(), principal.WorkshopID, orderID, principal.UserID)
	} else {
		token, err = h.service.repo.ManageLink(c.Request.Context(), principal.WorkshopID, orderID, principal.UserID, action)
	}
	if err != nil {
		if errors.Is(err, ErrTrackingRecoveryUnavailable) {
			httpx.RespondError(c, http.StatusConflict, "TRACKING_RECOVERY_UNAVAILABLE", "Replace the link explicitly to recover sharing after a server secret change.")
			return
		}
		if errors.Is(err, ErrOrderNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"token": token}})
}

func (h *TrackingHandler) Get(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	order, err := h.service.Find(c.Request.Context(), c.Param("token"))
	if err != nil {
		if errors.Is(err, ErrTrackingNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "TRACKING_NOT_FOUND", "This tracking link is not valid.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

func RegisterTrackingRoutes(routes gin.IRouter, handler *TrackingHandler, requireSession gin.HandlerFunc) {
	routes.POST("/api/v1/orders/:id/tracking-link", requireSession, handler.CreateLink)
	routes.POST("/api/v1/orders/:id/tracking-link/:action", requireSession, handler.CreateLink)
	routes.GET("/api/v1/tracking/:token", handler.Get)
}
