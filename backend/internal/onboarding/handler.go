package onboarding

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
	Status(context.Context) (Status, error)
	Create(context.Context, Input, auth.SessionMetadata) (Result, error)
}

type RequestLimiter interface {
	Allow(string) bool
}

type HandlerConfig struct {
	AllowedOrigin string
	Environment   string
	Secure        bool
	Limiter       RequestLimiter
}

type Handler struct {
	service    UseCases
	config     HandlerConfig
	cookieName string
}

func NewHandler(service UseCases, config HandlerConfig) (*Handler, error) {
	if service == nil || config.Limiter == nil {
		return nil, errors.New("onboarding service and limiter are required")
	}
	parsed, err := url.Parse(config.AllowedOrigin)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("allowed origin must be a scheme and host only")
	}
	if config.Environment == "production" && (parsed.Scheme != "https" || !config.Secure) {
		return nil, errors.New("production onboarding requires HTTPS cookies")
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

type createInput struct {
	WorkshopName         string `json:"workshop_name"`
	OwnerName            string `json:"owner_name"`
	Email                string `json:"email"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

func (h *Handler) Status(c *gin.Context) {
	if !h.allow(c) {
		return
	}
	status, err := h.service.Status(c.Request.Context())
	if err != nil {
		h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", gin.H{})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"available": status.Available}})
}

func (h *Handler) Create(c *gin.Context) {
	if !security.ValidRequestOrigin(c.Request, h.config.AllowedOrigin) {
		h.fail(c, http.StatusForbidden, "ORIGIN_INVALID", "Request origin is not allowed.", gin.H{})
		return
	}
	if !h.allow(c) {
		return
	}
	var body createInput
	if err := decodeJSON(c, &body, 2<<20+4096); err != nil {
		h.fail(c, http.StatusUnprocessableEntity, "ONBOARDING_VALIDATION_FAILED", "One or more fields are invalid.", gin.H{"request": "A single JSON object is required."})
		return
	}
	result, err := h.service.Create(c.Request.Context(), Input{
		WorkshopName: body.WorkshopName, OwnerName: body.OwnerName, Email: body.Email,
		Password: body.Password, PasswordConfirmation: body.PasswordConfirmation,
	}, auth.SessionMetadata{IPPrefix: strings.TrimSpace(c.ClientIP()), UserAgent: truncate(c.GetHeader("User-Agent"), 512)})
	if err != nil {
		var validation *ValidationError
		switch {
		case errors.As(err, &validation):
			h.fail(c, http.StatusUnprocessableEntity, "ONBOARDING_VALIDATION_FAILED", "One or more fields are invalid.", validation.Fields)
		case errors.Is(err, ErrUnavailable):
			h.fail(c, http.StatusConflict, "ONBOARDING_UNAVAILABLE", "This installation has already been claimed.", gin.H{})
		default:
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.", gin.H{})
		}
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cookieName, Value: result.Token, Path: "/", Secure: h.config.Secure,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: result.ExpiresAt.UTC(),
	})
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{
		"expires_at": result.ExpiresAt.UTC().Format(time.RFC3339), "csrf_token": result.CSRFToken,
		"must_change_password": result.MustChangePassword,
	}})
}

func (h *Handler) allow(c *gin.Context) bool {
	if h.config.Limiter.Allow(strings.TrimSpace(c.ClientIP())) {
		return true
	}
	c.Header("Retry-After", "900")
	h.fail(c, http.StatusTooManyRequests, "ONBOARDING_RATE_LIMITED", "Too many onboarding requests.", gin.H{})
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
