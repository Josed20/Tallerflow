package sprint2

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	bootstrap *gorm.DB
	tenant    database.TenantRunner
	hasher    auth.PasswordHasher
	now       func() time.Time
}

func New(db, bootstrap *gorm.DB, tenant database.TenantRunner) (*Handler, error) {
	if db == nil || tenant == nil {
		return nil, errors.New("database and tenant runner are required")
	}
	return &Handler{db: db, bootstrap: bootstrap, tenant: tenant, hasher: auth.NewPasswordHasher(auth.DefaultPasswordParams()), now: time.Now}, nil
}

func RegisterRoutes(r gin.IRouter, h *Handler, session gin.HandlerFunc, mutation gin.HandlerFunc) {
	r.GET("/api/v1/onboarding/status", h.onboardingStatus)
	r.POST("/api/v1/onboarding/workshop", h.onboard)
	r.POST("/api/v1/team/invitations/consume", h.consumeInvitation)
	p := r.Group("/api/v1", session)
	p.GET("/clients", h.listClients)
	p.GET("/orders", h.listOrders)
	p.GET("/dashboard", h.dashboard)
	p.GET("/team", h.listTeam)
	m := r.Group("/api/v1", mutation)
	m.POST("/clients", h.createClient)
	m.POST("/orders", h.createOrder)
	m.POST("/team/invitations", h.invite)
}

func (h *Handler) onboardingStatus(c *gin.Context) {
	if h.bootstrap == nil {
		httpx.RespondError(c, 503, "ONBOARDING_UNAVAILABLE", "Onboarding is not configured.")
		return
	}
	var n int64
	if err := h.bootstrap.WithContext(c).Raw("SELECT count(*) FROM bootstrap_state").Scan(&n).Error; err != nil {
		httpx.RespondError(c, 503, "DEPENDENCY_UNAVAILABLE", "A required dependency is unavailable.")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"available": n == 0}})
}

type onboardInput struct {
	WorkshopName string `json:"WorkshopName"`
	OwnerName    string `json:"OwnerName"`
	Email        string `json:"Email"`
	Password     string `json:"Password"`
}

func (h *Handler) onboard(c *gin.Context) {
	if h.bootstrap == nil {
		httpx.RespondError(c, 503, "ONBOARDING_UNAVAILABLE", "Onboarding is not configured.")
		return
	}
	var in onboardInput
	if c.ShouldBindJSON(&in) != nil || len(strings.TrimSpace(in.WorkshopName)) < 1 || len(strings.TrimSpace(in.OwnerName)) < 1 || !strings.Contains(in.Email, "@") || len(in.Password) < 12 {
		httpx.RespondError(c, 422, "ONBOARDING_INVALID", "Review the submitted fields.")
		return
	}
	hash, err := h.hasher.Hash(in.Password)
	if err != nil {
		httpx.RespondError(c, 422, "PASSWORD_INVALID", "The password is invalid.")
		return
	}
	err = h.bootstrap.WithContext(c).Transaction(func(tx *gorm.DB) error {
		var claimed int64
		if err := tx.Raw("SELECT count(*) FROM bootstrap_state").Scan(&claimed).Error; err != nil || claimed != 0 {
			if err != nil {
				return err
			}
			return errors.New("claimed")
		}
		wid, uid := uuid.New(), uuid.New()
		now := h.now().UTC()
		if err := tx.Exec("INSERT INTO workshops(id,name) VALUES (?,?)", wid, strings.TrimSpace(in.WorkshopName)).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO users(id,email,name) VALUES (?,lower(?),?)", uid, strings.TrimSpace(in.Email), strings.TrimSpace(in.OwnerName)).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO user_credentials(user_id,password_hash,must_change_password,password_changed_at) VALUES (?,?,false,?)", uid, hash, now).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO workshop_members(workshop_id,user_id,role) VALUES (?,?,'OWNER')", wid, uid).Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO audit_events(workshop_id,actor_user_id,event_type) VALUES (?,?,'WORKSHOP_ONBOARDED')", wid, uid).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO bootstrap_state(singleton,owner_user_id,bootstrapped_at,claimed_at,workshop_id) VALUES (true,?,?,?,?)", uid, now, now, wid).Error
	})
	if err != nil {
		httpx.RespondError(c, 409, "ONBOARDING_ALREADY_COMPLETED", "This installation has already been configured.")
		return
	}
	c.JSON(201, gin.H{"data": gin.H{"created": true}})
}

func principal(c *gin.Context) (httpx.Principal, bool) {
	p, e := httpx.PrincipalFromGin(c)
	if e != nil {
		httpx.RespondError(c, 401, "SESSION_INVALID", "The session is invalid or has expired.")
		return p, false
	}
	return p, true
}

type clientInput struct {
	Name  string `json:"Name"`
	Email string `json:"Email"`
	Phone string `json:"Phone"`
}

func (h *Handler) listClients(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var rows []map[string]any
	err := h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		return tx.Raw("SELECT id,name,email,phone,is_active FROM clients ORDER BY lower(name),id").Scan(&rows).Error
	})
	if err != nil {
		fail(c)
		return
	}
	c.JSON(200, gin.H{"data": rows})
}
func (h *Handler) createClient(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	if p.Role != "OWNER" && p.Role != "ADMIN" {
		httpx.RespondError(c, 403, "ROLE_FORBIDDEN", "The authenticated role cannot access this resource.")
		return
	}
	var in clientInput
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Name) == "" {
		httpx.RespondError(c, 422, "CLIENT_INVALID", "Review the submitted fields.")
		return
	}
	var row map[string]any
	err := h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		return tx.Raw("INSERT INTO clients(workshop_id,name,email,phone) VALUES (?,?,nullif(lower(trim(?)),''),nullif(trim(?),'')) RETURNING id,name,email,phone,is_active", p.WorkshopID, strings.TrimSpace(in.Name), in.Email, in.Phone).Scan(&row).Error
	})
	if err != nil {
		fail(c)
		return
	}
	c.JSON(201, gin.H{"data": row})
}

type orderInput struct {
	ClientID      uuid.UUID `json:"ClientID"`
	Product       string    `json:"Product"`
	Specification string    `json:"Specification"`
	Quantity      float64   `json:"Quantity"`
	Unit          string    `json:"Unit"`
	PromisedAt    time.Time `json:"PromisedAt"`
}

func (h *Handler) createOrder(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	if p.Role != "OWNER" && p.Role != "ADMIN" {
		httpx.RespondError(c, 403, "ROLE_FORBIDDEN", "The authenticated role cannot access this resource.")
		return
	}
	key := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	var in orderInput
	if key == "" || c.ShouldBindJSON(&in) != nil || in.ClientID == uuid.Nil || strings.TrimSpace(in.Product) == "" || in.Quantity <= 0 || in.PromisedAt.IsZero() {
		httpx.RespondError(c, 422, "ORDER_INVALID", "Review the submitted fields.")
		return
	}
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%g|%s|%s", in.ClientID, in.Product, in.Quantity, in.Unit, in.PromisedAt.UTC())))
	var row map[string]any
	err := h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		var existing struct {
			RequestHash []byte
			ResourceID  uuid.UUID
		}
		q := tx.Raw("SELECT request_hash,resource_id FROM idempotency_keys WHERE workshop_id=? AND key=?", p.WorkshopID, key).Scan(&existing)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 1 {
			if hex.EncodeToString(existing.RequestHash) != hex.EncodeToString(digest[:]) {
				return errors.New("idempotency")
			}
			return tx.Raw("SELECT id,code,product,quantity,unit,status,promised_at,version FROM orders WHERE id=?", existing.ResourceID).Scan(&row).Error
		}
		if err := tx.Exec("INSERT INTO workshop_counters(workshop_id,next_order_number) VALUES (?,1) ON CONFLICT DO NOTHING", p.WorkshopID).Error; err != nil {
			return err
		}
		var n int64
		if err := tx.Raw("UPDATE workshop_counters SET next_order_number=next_order_number+1 WHERE workshop_id=? RETURNING next_order_number-1", p.WorkshopID).Scan(&n).Error; err != nil {
			return err
		}
		id := uuid.New()
		code := fmt.Sprintf("OT-%06d", n)
		if err := tx.Exec("INSERT INTO orders(id,workshop_id,client_id,order_number,code,product,specification,quantity,unit,promised_at) VALUES (?,?,?,?,?,?,?,?,?,?)", id, p.WorkshopID, in.ClientID, n, code, strings.TrimSpace(in.Product), in.Specification, in.Quantity, in.Unit, in.PromisedAt.UTC()).Error; err != nil {
			return err
		}
		stages := []string{"CORTE", "COSTURA", "ACABADO", "CONTROL DE CALIDAD", "PLANCHADO", "EMPAQUE", "ENTREGA"}
		for i, name := range stages {
			status := "PENDING"
			if i == 0 {
				status = "IN_PROGRESS"
			}
			if err := tx.Exec("INSERT INTO order_stages(workshop_id,order_id,position,name,status) VALUES (?,?,?,?,?)", p.WorkshopID, id, i+1, name, status).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("INSERT INTO idempotency_keys(workshop_id,key,request_hash,resource_id) VALUES (?,?,?,?)", p.WorkshopID, key, digest[:], id).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT id,code,product,quantity,unit,status,promised_at,version FROM orders WHERE id=?", id).Scan(&row).Error
	})
	if err != nil {
		if err.Error() == "tenant transaction: idempotency" {
			httpx.RespondError(c, 409, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used with another payload.")
			return
		}
		fail(c)
		return
	}
	c.JSON(201, gin.H{"data": row})
}

func (h *Handler) listOrders(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var rows []map[string]any
	now := h.now().UTC()
	err := h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		return tx.Raw(`SELECT o.id,o.code,o.product,c.name client_name,o.status,o.promised_at,CASE WHEN o.status='COMPLETED' THEN 'COMPLETED' WHEN o.promised_at < ? THEN 'LATE' WHEN o.promised_at < ? THEN 'AT_RISK' ELSE 'ON_TIME' END traffic_light FROM orders o JOIN clients c ON c.id=o.client_id ORDER BY o.created_at DESC,o.id DESC LIMIT 100`, now, now.Add(48*time.Hour)).Scan(&rows).Error
	})
	if err != nil {
		fail(c)
		return
	}
	c.JSON(200, gin.H{"data": rows})
}
func (h *Handler) dashboard(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	if p.Role != "OWNER" && p.Role != "ADMIN" {
		httpx.RespondError(c, 403, "ROLE_FORBIDDEN", "The authenticated role cannot access this resource.")
		return
	}
	var d struct{ Active, AtRisk, Late int64 }
	now := h.now().UTC()
	err := h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		return tx.Raw(`SELECT count(*) FILTER(WHERE status='ACTIVE') active,count(*) FILTER(WHERE status='ACTIVE' AND promised_at>=? AND promised_at<?) at_risk,count(*) FILTER(WHERE status='ACTIVE' AND promised_at<?) late FROM orders`, now, now.Add(48*time.Hour), now).Scan(&d).Error
	})
	if err != nil {
		fail(c)
		return
	}
	c.JSON(200, gin.H{"data": d})
}

func (h *Handler) listTeam(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var rows []map[string]any
	err := h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		return tx.Raw("SELECT u.id,u.name display_name,u.email::text email,wm.role,wm.status FROM workshop_members wm JOIN users u ON u.id=wm.user_id ORDER BY wm.created_at").Scan(&rows).Error
	})
	if err != nil {
		fail(c)
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

type inviteInput struct {
	Email string `json:"Email"`
	Role  string `json:"Role"`
}

func (h *Handler) invite(c *gin.Context) {
	p, ok := principal(c)
	if !ok {
		return
	}
	var in inviteInput
	if c.ShouldBindJSON(&in) != nil || (in.Role != "ADMIN" && in.Role != "OPERATOR") || p.Role == "OPERATOR" || (p.Role == "ADMIN" && in.Role != "OPERATOR") {
		httpx.RespondError(c, 403, "ROLE_FORBIDDEN", "The authenticated role cannot create this invitation.")
		return
	}
	token, digest, err := newToken()
	if err != nil {
		fail(c)
		return
	}
	err = h.tenant.WithinTenant(c, p.WorkshopID, func(tx *gorm.DB) error {
		return tx.Exec("INSERT INTO team_invitations(workshop_id,email,role,token_hash,expires_at,created_by) VALUES (?,lower(trim(?)),?,?,?,?)", p.WorkshopID, in.Email, in.Role, digest, h.now().UTC().Add(48*time.Hour), p.UserID).Error
	})
	if err != nil {
		httpx.RespondError(c, 409, "INVITATION_CONFLICT", "An active invitation already exists.")
		return
	}
	c.JSON(201, gin.H{"data": gin.H{"invitationUrl": "/join?token=" + token}})
}

type consumeInviteInput struct{ Token, Name, Password string }

func (h *Handler) consumeInvitation(c *gin.Context) {
	var in consumeInviteInput
	if c.ShouldBindJSON(&in) != nil || len(in.Password) < 12 {
		httpx.RespondError(c, 422, "INVITATION_INVALID", "The invitation is invalid or expired.")
		return
	}
	digest := sha256.Sum256([]byte(in.Token))
	hash, err := h.hasher.Hash(in.Password)
	if err != nil {
		fail(c)
		return
	}
	if h.bootstrap == nil {
		httpx.RespondError(c, 503, "INVITATION_UNAVAILABLE", "Invitation acceptance is not configured.")
		return
	}
	err = h.bootstrap.WithContext(c).Transaction(func(tx *gorm.DB) error {
		var v struct {
			ID, WorkshopID uuid.UUID
			Email, Role    string
			ExpiresAt      time.Time
		}
		q := tx.Raw("SELECT id,workshop_id,email::text email,role,expires_at FROM team_invitations WHERE token_hash=? AND consumed_at IS NULL FOR UPDATE", digest[:]).Scan(&v)
		if q.Error != nil || q.RowsAffected != 1 || !v.ExpiresAt.After(h.now()) {
			return errors.New("invalid")
		}
		var uid uuid.UUID
		q = tx.Raw("SELECT id FROM users WHERE email=?", v.Email).Scan(&uid)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			uid = uuid.New()
			if err := tx.Exec("INSERT INTO users(id,email,name) VALUES (?,?,?)", uid, v.Email, strings.TrimSpace(in.Name)).Error; err != nil {
				return err
			}
			if err := tx.Exec("INSERT INTO user_credentials(user_id,password_hash,must_change_password,password_changed_at) VALUES (?,?,false,?)", uid, hash, h.now().UTC()).Error; err != nil {
				return err
			}
		}
		if err := tx.Exec("INSERT INTO workshop_members(workshop_id,user_id,role) VALUES (?,?,?)", v.WorkshopID, uid, v.Role).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE team_invitations SET consumed_at=? WHERE id=?", h.now().UTC(), v.ID).Error
	})
	if err != nil {
		httpx.RespondError(c, 422, "INVITATION_INVALID", "The invitation is invalid or expired.")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"accepted": true}})
}

func newToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw := hex.EncodeToString(b)
	d := sha256.Sum256([]byte(raw))
	return raw, d[:], nil
}
func fail(c *gin.Context) {
	httpx.RespondError(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
}
