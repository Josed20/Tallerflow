package signup

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UseCases interface {
	Register(context.Context, Input) error
	Resend(context.Context, string) error
	Verify(context.Context, string, auth.SessionMetadata) (auth.RawSession, error)
}

type RequestLimiter interface {
	Allow(string) bool
}

type HandlerConfig struct {
	AllowedOrigin string
	Environment   string
	Secure        bool
	// Each endpoint has its own per-IP budget, so a burst of resends cannot
	// lock a visitor out of verifying.
	RegisterLimiter RequestLimiter
	ResendLimiter   RequestLimiter
	VerifyLimiter   RequestLimiter
}

type Handler struct {
	service    UseCases
	config     HandlerConfig
	cookieName string
}

func NewHandler(service UseCases, config HandlerConfig) (*Handler, error) {
	if service == nil || config.RegisterLimiter == nil || config.ResendLimiter == nil || config.VerifyLimiter == nil {
		return nil, errors.New("signup service and limiters are required")
	}
	parsed, err := url.Parse(config.AllowedOrigin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("allowed origin must be a scheme and host only")
	}
	if config.Environment == "production" && (parsed.Scheme != "https" || !config.Secure) {
		return nil, errors.New("production signup requires HTTPS cookies")
	}
	cookieName := auth.SessionCookieName
	if !config.Secure {
		if config.Environment != "development" {
			return nil, errors.New("insecure session cookie is permitted only in development")
		}
		cookieName = "tallerflow_session"
	}
	return &Handler{service: service, config: config, cookieName: cookieName}, nil
}

type registerInput struct {
	WorkshopName string `json:"workshop_name"`
	OwnerName    string `json:"owner_name"`
	Email        string `json:"email"`
	Password     string `json:"password"`
	AcceptTerms  bool   `json:"accept_terms"`
}

type tokenInput struct {
	Token string `json:"token"`
}

type emailInput struct {
	Email string `json:"email"`
}

func (h *Handler) Register(c *gin.Context) {
	if !security.ValidRequestOrigin(c.Request, h.config.AllowedOrigin) {
		h.fail(c, http.StatusForbidden, "ORIGIN_INVALID", "Request origin is not allowed.", gin.H{})
		return
	}
	if !h.allow(c, h.config.RegisterLimiter) {
		return
	}
	var body registerInput
	if err := decodeJSON(c, &body, 1<<20+4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.", gin.H{})
		return
	}
	err := h.service.Register(c.Request.Context(), Input{
		WorkshopName: body.WorkshopName, OwnerName: body.OwnerName, Email: body.Email,
		Password: body.Password, AcceptTerms: body.AcceptTerms,
	})
	var validation *ValidationError
	switch {
	case err == nil:
		c.JSON(http.StatusAccepted, gin.H{"data": gin.H{}})
	case errors.Is(err, ErrClosed):
		h.fail(c, http.StatusForbidden, "SIGNUP_CLOSED", "Registration is temporarily closed.", gin.H{})
	case errors.As(err, &validation):
		h.fail(c, http.StatusUnprocessableEntity, "SIGNUP_VALIDATION_FAILED", "One or more fields are invalid.", gin.H{"fields": validation.Fields})
	default:
		h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", gin.H{})
	}
}

func (h *Handler) Resend(c *gin.Context) {
	if !security.ValidRequestOrigin(c.Request, h.config.AllowedOrigin) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.", gin.H{})
		return
	}
	if !h.allow(c, h.config.ResendLimiter) {
		return
	}
	var body emailInput
	if err := decodeJSON(c, &body, 4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.", gin.H{})
		return
	}
	err := h.service.Resend(c.Request.Context(), body.Email)
	var validation *ValidationError
	switch {
	case err == nil:
		c.JSON(http.StatusAccepted, gin.H{"data": gin.H{}})
	case errors.As(err, &validation):
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.", gin.H{})
	default:
		h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", gin.H{})
	}
}

func (h *Handler) Verify(c *gin.Context) {
	if !security.ValidRequestOrigin(c.Request, h.config.AllowedOrigin) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.", gin.H{})
		return
	}
	if !h.allow(c, h.config.VerifyLimiter) {
		return
	}
	var body tokenInput
	if err := decodeJSON(c, &body, 4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.", gin.H{})
		return
	}
	session, err := h.service.Verify(c.Request.Context(), body.Token, auth.SessionMetadata{
		IPPrefix: strings.TrimSpace(c.ClientIP()), UserAgent: truncate(c.GetHeader("User-Agent"), 512),
	})
	switch {
	case errors.Is(err, ErrTokenInvalid):
		h.fail(c, http.StatusUnprocessableEntity, "VERIFICATION_TOKEN_INVALID", "The verification link is invalid or has expired.", gin.H{})
		return
	case err != nil:
		h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", gin.H{})
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cookieName, Value: session.Token, Path: "/", Secure: h.config.Secure,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: session.ExpiresAt.UTC(),
	})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"expires_at": session.ExpiresAt.UTC().Format(time.RFC3339), "csrf_token": session.CSRFToken,
		"must_change_password": false,
	}})
}

func (h *Handler) allow(c *gin.Context, limiter RequestLimiter) bool {
	if limiter.Allow(strings.TrimSpace(c.ClientIP())) {
		return true
	}
	c.Header("Retry-After", "900")
	h.fail(c, http.StatusTooManyRequests, "SIGNUP_RATE_LIMITED", "Too many requests. Try again later.", gin.H{})
	return false
}

func (h *Handler) fail(c *gin.Context, status int, code, message string, details any) {
	requestID := httpx.RequestIDFromContext(c)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.Header("X-Request-ID", requestID)
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message, "details": details, "request_id": requestID}})
}

func decodeJSON(c *gin.Context, target any, maxBytes int64) error {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("JSON content type required")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
