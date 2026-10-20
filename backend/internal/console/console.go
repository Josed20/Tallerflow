package console

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const cookieName = "tallerflow_console"

type Admin struct {
	ID                   uuid.UUID
	Email                string
	PasswordHash         string
	TOTPSecretCiphertext []byte
	Status               string
}
type Session struct {
	ID         uuid.UUID
	AdminID    uuid.UUID
	ExpiresAt  time.Time
	LastSeenAt time.Time
}
type Workshop struct {
	ID                   uuid.UUID  `json:"id"`
	Name                 string     `json:"name"`
	OwnerEmail           string     `json:"owner_email"`
	CreatedAt            time.Time  `json:"created_at"`
	LastActivity         *time.Time `json:"last_activity"`
	Members              int64      `json:"members"`
	ActiveOrders         int64      `json:"active_orders"`
	PhotoBytes           int64      `json:"photo_bytes"`
	Status               string     `json:"status"`
	DeletionScheduledFor *time.Time `json:"deletion_scheduled_for"`
}
type AuditEvent struct {
	ID         uuid.UUID  `json:"id"`
	WorkshopID *uuid.UUID `json:"workshop_id"`
	EventType  string     `json:"event_type"`
	Reason     string     `json:"reason"`
	SourceIP   string     `json:"source_ip"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Service struct {
	db     *gorm.DB
	hasher auth.PasswordHasher
	key    []byte
	now    func() time.Time
	random io.Reader
}

func NewService(db *gorm.DB, encryptionKey string) (*Service, error) {
	if db == nil {
		return nil, errors.New("console database is required")
	}
	if strings.TrimSpace(encryptionKey) == "" {
		return nil, errors.New("console TOTP key is required")
	}
	digest := sha256.Sum256([]byte(encryptionKey))
	return &Service{db: db, hasher: auth.NewPasswordHasher(auth.DefaultPasswordParams()), key: digest[:], now: time.Now, random: rand.Reader}, nil
}

func EncryptSecret(keyText, secret string) ([]byte, error) {
	key := sha256.Sum256([]byte(keyText))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(secret), nil), nil
}

func GenerateSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

func (s *Service) decrypt(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("invalid TOTP secret")
	}
	plain, err := gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], nil)
	return string(plain), err
}

func validTOTP(secret, code string, now time.Time) bool {
	secretBytes, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return false
	}
	for offset := int64(-1); offset <= 1; offset++ {
		counter := uint64(now.Unix()/30 + offset)
		payload := make([]byte, 8)
		binary.BigEndian.PutUint64(payload, counter)
		mac := hmac.New(sha1.New, secretBytes)
		_, _ = mac.Write(payload)
		sum := mac.Sum(nil)
		pos := sum[len(sum)-1] & 15
		value := (uint32(sum[pos])&0x7f)<<24 | uint32(sum[pos+1])<<16 | uint32(sum[pos+2])<<8 | uint32(sum[pos+3])
		if hmac.Equal([]byte(fmt.Sprintf("%06d", value%1000000)), []byte(strings.TrimSpace(code))) {
			return true
		}
	}
	return false
}

func tokenDigest(raw string) []byte { d := sha256.Sum256([]byte(raw)); return d[:] }
func randomToken(reader io.Reader) (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) Login(ctx context.Context, email, password, code string) (string, string, time.Time, error) {
	var admin Admin
	q := s.db.WithContext(ctx).Raw("SELECT id, email, password_hash, totp_secret_ciphertext, status FROM console_admins WHERE email = ?", strings.ToLower(strings.TrimSpace(email))).Scan(&admin)
	if q.Error != nil || q.RowsAffected == 0 || admin.Status != "ACTIVE" {
		return "", "", time.Time{}, errors.New("invalid credentials")
	}
	ok, err := s.hasher.Verify(admin.PasswordHash, password)
	if err != nil || !ok {
		return "", "", time.Time{}, errors.New("invalid credentials")
	}
	secret, err := s.decrypt(admin.TOTPSecretCiphertext)
	if err != nil || !validTOTP(secret, code, s.now()) {
		return "", "", time.Time{}, errors.New("invalid credentials")
	}
	token, err := randomToken(s.random)
	if err != nil {
		return "", "", time.Time{}, err
	}
	csrf, err := randomToken(s.random)
	if err != nil {
		return "", "", time.Time{}, err
	}
	now := s.now().UTC()
	expires := now.Add(8 * time.Hour)
	csrfHash := sha256.Sum256([]byte(csrf))
	err = s.db.WithContext(ctx).Exec("INSERT INTO console_sessions (id, admin_id, token_hash, csrf_token_hash, expires_at, last_seen_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)", uuid.New(), admin.ID, tokenDigest(token), csrfHash[:], expires, now, now).Error
	return token, csrf, expires, err
}

func (s *Service) Authenticate(ctx context.Context, token string) (Session, error) {
	var session Session
	now := s.now().UTC()
	q := s.db.WithContext(ctx).Raw("SELECT id, admin_id, expires_at, last_seen_at FROM console_sessions WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > ? AND last_seen_at > ?", tokenDigest(token), now, now.Add(-30*time.Minute)).Scan(&session)
	if q.Error != nil || q.RowsAffected == 0 {
		return Session{}, errors.New("invalid console session")
	}
	if err := s.db.WithContext(ctx).Exec("UPDATE console_sessions SET last_seen_at = ? WHERE id = ?", now, session.ID).Error; err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) ValidateCSRF(ctx context.Context, sessionID uuid.UUID, raw string) bool {
	var stored []byte
	if s.db.WithContext(ctx).Raw("SELECT csrf_token_hash FROM console_sessions WHERE id = ?", sessionID).Scan(&stored).Error != nil {
		return false
	}
	digest := sha256.Sum256([]byte(raw))
	return hmac.Equal(stored, digest[:])
}
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	return s.db.WithContext(ctx).Exec("UPDATE console_sessions SET revoked_at = ? WHERE id = ?", s.now().UTC(), sessionID).Error
}
func (s *Service) ListWorkshops(ctx context.Context, search string, limit, offset int) ([]Workshop, error) {
	var rows []Workshop
	err := s.db.WithContext(ctx).Raw("SELECT * FROM console_list_workshops(?, ?, ?)", strings.TrimSpace(search), limit, offset).Scan(&rows).Error
	return rows, err
}
func (s *Service) SetStatus(ctx context.Context, adminID, workshopID uuid.UUID, status, reason string) error {
	return s.db.WithContext(ctx).Exec("SELECT console_set_workshop_status(?, ?, ?, ?)", workshopID, strings.ToUpper(status), strings.TrimSpace(reason), adminID).Error
}
func (s *Service) VerifyTOTP(ctx context.Context, adminID uuid.UUID, code string) bool {
	var ciphertext []byte
	if s.db.WithContext(ctx).Raw("SELECT totp_secret_ciphertext FROM console_admins WHERE id = ? AND status = 'ACTIVE'", adminID).Scan(&ciphertext).Error != nil || len(ciphertext) == 0 {
		return false
	}
	secret, err := s.decrypt(ciphertext)
	return err == nil && validTOTP(secret, code, s.now())
}
func (s *Service) Purge(ctx context.Context, adminID, workshopID uuid.UUID, name string) error {
	return s.db.WithContext(ctx).Exec("SELECT console_purge_workshop(?, ?, ?)", workshopID, name, adminID).Error
}
func (s *Service) Audit(ctx context.Context, limit int) ([]AuditEvent, error) {
	var rows []AuditEvent
	err := s.db.WithContext(ctx).Raw("SELECT id, workshop_id, event_type, reason, source_ip, created_at FROM console_audit_events ORDER BY created_at DESC LIMIT ?", limit).Scan(&rows).Error
	return rows, err
}

type Handler struct {
	service *Service
	origin  string
	secure  bool
	limiter *security.MemoryRateLimiter
}

func NewHandler(service *Service, origin string, secure bool) *Handler {
	return &Handler{service: service, origin: origin, secure: secure, limiter: security.NewMemoryRateLimiter([]byte("console-login:"+origin), time.Now, 15*time.Minute, 10, 10000)}
}
func (h *Handler) cookieName() string {
	if h.secure {
		return "__Host-tallerflow_console"
	}
	return cookieName
}
func (h *Handler) setCookie(c *gin.Context, value string, maxAge int) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(h.cookieName(), value, maxAge, "/", "", h.secure, true)
}
func (h *Handler) sameOrigin(c *gin.Context) bool {
	origin := c.GetHeader("Origin")
	return origin == "" || origin == h.origin
}

func (h *Handler) Login(c *gin.Context) {
	if !h.sameOrigin(c) {
		httpx.RespondError(c, 403, "ORIGIN_INVALID", "Request origin is not allowed.")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Code     string `json:"code"`
	}
	if c.ShouldBindJSON(&in) != nil {
		httpx.RespondError(c, 400, "INVALID_REQUEST", "Review the submitted fields.")
		return
	}
	if !h.limiter.Allow(strings.TrimSpace(c.ClientIP()) + "|" + strings.ToLower(strings.TrimSpace(in.Email))) {
		c.Header("Retry-After", "900")
		httpx.RespondError(c, http.StatusTooManyRequests, "CONSOLE_RATE_LIMITED", "Too many login attempts.")
		return
	}
	token, csrf, expires, err := h.service.Login(c, in.Email, in.Password, in.Code)
	if err != nil {
		httpx.RespondError(c, 401, "CONSOLE_INVALID_CREDENTIALS", "Email, password or verification code is incorrect.")
		return
	}
	h.setCookie(c, token, int(time.Until(expires).Seconds()))
	c.JSON(200, gin.H{"data": gin.H{"csrf_token": csrf, "expires_at": expires}})
}
func (h *Handler) RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(h.cookieName())
		if err != nil {
			httpx.RespondError(c, 401, "CONSOLE_SESSION_INVALID", "The console session is invalid or expired.")
			c.Abort()
			return
		}
		session, err := h.service.Authenticate(c, token)
		if err != nil {
			h.setCookie(c, "", -1)
			httpx.RespondError(c, 401, "CONSOLE_SESSION_INVALID", "The console session is invalid or expired.")
			c.Abort()
			return
		}
		c.Set("console_session", session)
		c.Next()
	}
}
func consoleSession(c *gin.Context) Session {
	value, _ := c.Get("console_session")
	session, _ := value.(Session)
	return session
}
func (h *Handler) requireMutation(c *gin.Context) bool {
	session := consoleSession(c)
	if !h.sameOrigin(c) || !h.service.ValidateCSRF(c, session.ID, c.GetHeader("X-CSRF-Token")) {
		httpx.RespondError(c, 403, "CSRF_INVALID", "The security token is invalid.")
		return false
	}
	return true
}
func (h *Handler) Session(c *gin.Context) {
	session := consoleSession(c)
	c.JSON(200, gin.H{"data": gin.H{"admin_id": session.AdminID, "expires_at": session.ExpiresAt}})
}
func (h *Handler) Logout(c *gin.Context) {
	if !h.requireMutation(c) {
		return
	}
	_ = h.service.Logout(c, consoleSession(c).ID)
	h.setCookie(c, "", -1)
	c.JSON(200, gin.H{"data": gin.H{}})
}
func (h *Handler) Workshops(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	rows, err := h.service.ListWorkshops(c, c.Query("q"), limit, offset)
	if err != nil {
		httpx.RespondError(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	c.JSON(200, gin.H{"data": rows})
}
func (h *Handler) SetStatus(c *gin.Context) {
	if !h.requireMutation(c) {
		return
	}
	workshopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, 404, "WORKSHOP_NOT_FOUND", "The workshop does not exist.")
		return
	}
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if c.ShouldBindJSON(&in) != nil || len([]rune(strings.TrimSpace(in.Reason))) < 5 {
		httpx.RespondError(c, 422, "REASON_INVALID", "A reason of at least 5 characters is required.")
		return
	}
	if err := h.service.SetStatus(c, consoleSession(c).AdminID, workshopID, in.Status, in.Reason); err != nil {
		httpx.RespondError(c, 409, "WORKSHOP_STATUS_INVALID", "The workshop status could not be changed.")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{}})
}
func (h *Handler) Purge(c *gin.Context) {
	if !h.requireMutation(c) {
		return
	}
	workshopID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, 404, "WORKSHOP_NOT_FOUND", "The workshop does not exist.")
		return
	}
	var in struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if c.ShouldBindJSON(&in) != nil {
		httpx.RespondError(c, 422, "CONFIRMATION_INVALID", "The workshop name is required.")
		return
	}
	if !h.service.VerifyTOTP(c, consoleSession(c).AdminID, in.Code) {
		httpx.RespondError(c, http.StatusForbidden, "TOTP_INVALID", "A new verification code is required.")
		return
	}
	if err := h.service.Purge(c, consoleSession(c).AdminID, workshopID, in.Name); err != nil {
		httpx.RespondError(c, 409, "WORKSHOP_PURGE_BLOCKED", "The workshop cannot be deleted yet or its name does not match.")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{}})
}
func (h *Handler) Audit(c *gin.Context) {
	rows, err := h.service.Audit(c, 100)
	if err != nil {
		httpx.RespondError(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	c.JSON(200, gin.H{"data": rows})
}

func RegisterRoutes(routes gin.IRouter, h *Handler) {
	authn := h.RequireSession()
	routes.POST("/api/v1/console/login", h.Login)
	routes.GET("/api/v1/console/session", authn, h.Session)
	routes.POST("/api/v1/console/logout", authn, h.Logout)
	routes.GET("/api/v1/console/workshops", authn, h.Workshops)
	routes.POST("/api/v1/console/workshops/:id/status", authn, h.SetStatus)
	routes.POST("/api/v1/console/workshops/:id/purge", authn, h.Purge)
	routes.GET("/api/v1/console/audit", authn, h.Audit)
}
