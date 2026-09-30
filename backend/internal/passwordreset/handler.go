package passwordreset

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	delivery  PasswordResetDelivery
	publicURL string
	hasher    auth.PasswordHasher
	now       func() time.Time
}

func New(db *gorm.DB, delivery PasswordResetDelivery, publicURL string) (*Handler, error) {
	if db == nil || delivery == nil {
		return nil, errors.New("database and password reset delivery are required")
	}
	base, err := url.Parse(strings.TrimRight(publicURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, errors.New("password reset public URL is invalid")
	}
	return &Handler{db: db, delivery: delivery, publicURL: base.String(), hasher: auth.NewPasswordHasher(auth.DefaultPasswordParams()), now: time.Now}, nil
}

func RegisterRoutes(r gin.IRouter, h *Handler) {
	r.POST("/api/v1/auth/password-resets", h.request)
	r.POST("/api/v1/auth/password-resets/consume", h.consume)
}

type requestInput struct {
	Email string `json:"Email"`
}
type consumeInput struct {
	Token    string `json:"Token"`
	Password string `json:"Password"`
}

func (h *Handler) request(c *gin.Context) {
	var in requestInput
	_ = c.ShouldBindJSON(&in)
	var user struct {
		ID    uuid.UUID
		Email string
	}
	q := h.db.WithContext(c).Raw("SELECT id,email::text email FROM users WHERE email=lower(trim(?)) AND status='ACTIVE'", in.Email).Scan(&user)
	if q.Error == nil && q.RowsAffected == 1 {
		raw, digest, err := newToken()
		if err == nil {
			err = h.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
				if err := tx.Exec("UPDATE password_reset_tokens SET used_at=? WHERE user_id=? AND used_at IS NULL", h.now().UTC(), user.ID).Error; err != nil {
					return err
				}
				return tx.Exec("INSERT INTO password_reset_tokens(user_id,token_hash,expires_at) VALUES (?,?,?)", user.ID, digest, h.now().UTC().Add(time.Hour)).Error
			})
			if err == nil {
				resetURL := h.publicURL + "/reset-password?token=" + url.QueryEscape(raw)
				_ = h.delivery.DeliverPasswordReset(c, user.Email, resetURL)
			}
		}
	}
	// Deliberately uniform: account existence and infrastructure failures are private.
	c.JSON(202, gin.H{"data": gin.H{"accepted": true}})
}

func (h *Handler) consume(c *gin.Context) {
	var in consumeInput
	if c.ShouldBindJSON(&in) != nil || strings.TrimSpace(in.Token) == "" || len(in.Password) < 12 {
		httpx.RespondError(c, 422, "RESET_INVALID", "The reset token is invalid, expired, or already used.")
		return
	}
	digest := sha256.Sum256([]byte(in.Token))
	hash, err := h.hasher.Hash(in.Password)
	if err != nil {
		httpx.RespondError(c, 422, "RESET_INVALID", "The reset token is invalid, expired, or already used.")
		return
	}
	err = h.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		var token struct {
			ID, UserID uuid.UUID
			ExpiresAt  time.Time
		}
		q := tx.Raw("SELECT id,user_id,expires_at FROM password_reset_tokens WHERE token_hash=? AND used_at IS NULL FOR UPDATE", digest[:]).Scan(&token)
		if q.Error != nil || q.RowsAffected != 1 || !token.ExpiresAt.After(h.now()) {
			return errors.New("invalid reset")
		}
		now := h.now().UTC()
		if err := tx.Exec("UPDATE user_credentials SET password_hash=?,must_change_password=false,password_changed_at=?,updated_at=? WHERE user_id=?", hash, now, now, token.UserID).Error; err != nil {
			return err
		}
		if err := tx.Exec("UPDATE user_sessions SET revoked_at=? WHERE user_id=? AND revoked_at IS NULL", now, token.UserID).Error; err != nil {
			return err
		}
		return tx.Exec("UPDATE password_reset_tokens SET used_at=? WHERE id=?", now, token.ID).Error
	})
	if err != nil {
		httpx.RespondError(c, 422, "RESET_INVALID", "The reset token is invalid, expired, or already used.")
		return
	}
	c.JSON(200, gin.H{"data": gin.H{"changed": true}})
}

func newToken() (string, []byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw := base64.RawURLEncoding.EncodeToString(b)
	digest := sha256.Sum256([]byte(raw))
	return raw, digest[:], nil
}
