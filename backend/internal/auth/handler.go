package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const SessionCookieName = "__Host-tallerflow_session"

type AuthUseCases interface {
	Login(context.Context, string, string, string, SessionKind) (AuthSession, error)
	Restore(context.Context, string) (AuthSession, error)
	Logout(context.Context, string) error
	ChangePassword(context.Context, string, string, string) (AuthSession, error)
	EnsureWorkshopActive(context.Context, uuid.UUID) error
}

// HandlerConfig separates the two transports: the browser (cookie + CSRF)
// must come from AllowedOrigin, while the Android app (bearer token) comes
// from one of MobileOrigins or from a native client that sends no Origin.
type HandlerConfig struct {
	AllowedOrigin string
	MobileOrigins []string
	Environment   string
	Secure        bool
}

type Handler struct {
	service    AuthUseCases
	config     HandlerConfig
	cookieName string
}

func NewHandler(service AuthUseCases, config HandlerConfig) (*Handler, error) {
	if service == nil {
		return nil, errors.New("auth service is required")
	}
	if strings.TrimSpace(config.AllowedOrigin) == "" {
		return nil, errors.New("allowed origin is required")
	}
	parsedOrigin, err := url.Parse(config.AllowedOrigin)
	if err != nil || (parsedOrigin.Scheme != "http" && parsedOrigin.Scheme != "https") || parsedOrigin.Host == "" || parsedOrigin.User != nil || parsedOrigin.Path != "" || parsedOrigin.RawQuery != "" || parsedOrigin.Fragment != "" {
		return nil, errors.New("allowed origin must be a scheme and host only")
	}
	if config.Environment == "production" && parsedOrigin.Scheme != "https" {
		return nil, errors.New("production allowed origin must use HTTPS")
	}
	if !config.Secure && config.Environment != "development" {
		return nil, errors.New("insecure session cookie is permitted only in development")
	}
	cookieName := SessionCookieName
	if !config.Secure {
		cookieName = "tallerflow_session"
	}
	return &Handler{service: service, config: config, cookieName: cookieName}, nil
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required,max=1048576"`
	Client   string `json:"client"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	if h.hasCookie(c) && c.GetHeader("Authorization") != "" {
		h.ambiguousCredentials(c)
		return
	}
	var input LoginInput
	if err := decodeJSON(c, &input, 1<<20+4096); err != nil || !validEmail(input.Email) || input.Password == "" || len(input.Password) > 1<<20 {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
		return
	}
	kind := SessionKindWeb
	switch input.Client {
	case "", "web":
		if !security.ValidRequestOrigin(c.Request, h.config.AllowedOrigin) {
			h.fail(c, http.StatusForbidden, "ORIGIN_INVALID", "Request origin is not allowed.")
			return
		}
	case "mobile":
		if !h.mobileOriginAllowed(c) {
			h.fail(c, http.StatusForbidden, "ORIGIN_INVALID", "Request origin is not allowed.")
			return
		}
		kind = SessionKindMobile
	default:
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
		return
	}
	result, err := h.service.Login(c.Request.Context(), input.Email, input.Password, requestIP(c), kind)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			h.fail(c, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Email or password is incorrect.")
		case errors.Is(err, ErrEmailNotVerified):
			h.fail(c, http.StatusForbidden, "EMAIL_NOT_VERIFIED", "Verify your email before signing in.")
		case errors.Is(err, ErrWorkshopSuspended):
			h.workshopSuspended(c)
		case errors.Is(err, ErrTooManyAttempts):
			c.Header("Retry-After", "900")
			h.fail(c, http.StatusTooManyRequests, "AUTH_RATE_LIMITED", "Too many login attempts.")
		default:
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return
	}
	h.respondSession(c, result, true)
}

func (h *Handler) Session(c *gin.Context) {
	result, ok := h.restore(c)
	if !ok || !h.workshopActive(c, result) {
		return
	}
	h.respondSession(c, result, false)
}

func (h *Handler) Logout(c *gin.Context) {
	result, ok := h.restore(c)
	if !ok {
		return
	}
	if result.Kind == SessionKindWeb && !h.requireCSRF(c, result) {
		return
	}
	if err := h.service.Logout(c.Request.Context(), result.Token); err != nil {
		if errors.Is(err, ErrSessionInvalid) {
			h.invalidSession(c)
		} else {
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return
	}
	if result.Kind == SessionKindWeb {
		h.clearCookie(c)
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{}})
}

func (h *Handler) ChangePassword(c *gin.Context) {
	current, ok := h.restore(c)
	if !ok {
		return
	}
	if current.Kind == SessionKindWeb && !h.requireCSRF(c, current) {
		return
	}
	var input ChangePasswordInput
	if err := decodeJSON(c, &input, 2<<20+4096); err != nil || input.CurrentPassword == "" || input.NewPassword == "" || len(input.CurrentPassword) > 1<<20 || len(input.NewPassword) > 1<<20 {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
		return
	}
	replacement, err := h.service.ChangePassword(c.Request.Context(), current.Token, input.CurrentPassword, input.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, ErrSessionInvalid):
			h.invalidSession(c)
		case errors.Is(err, ErrPasswordInvalid), errors.Is(err, ErrPasswordPolicy), errors.Is(err, ErrPasswordReuse):
			h.fail(c, http.StatusUnprocessableEntity, "PASSWORD_INVALID", "Current or new password is invalid.")
		default:
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return
	}
	h.respondSession(c, replacement, true)
}

// respondSession answers with the browser shape (cookie + CSRF token) or the
// mobile shape (bearer token in JSON, no cookie). The bearer token is only
// returned when it was just issued.
func (h *Handler) respondSession(c *gin.Context, session AuthSession, issued bool) {
	if session.Kind == SessionKindMobile {
		data := gin.H{
			"expires_at":           session.ExpiresAt.UTC().Format(time.RFC3339),
			"must_change_password": session.MustChangePassword,
		}
		if issued {
			data["session_token"] = session.Token
		}
		c.JSON(http.StatusOK, gin.H{"data": data})
		return
	}
	if issued {
		h.setCookie(c, session.Token, session.ExpiresAt)
	}
	c.JSON(http.StatusOK, gin.H{"data": sessionData(session)})
}

// RequireSession protects private routes and enforces the suspended workshop
// and bootstrap password gates. Logout and the password change call restore
// directly and remain usable.
func (h *Handler) RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		result, ok := h.restore(c)
		if !ok || !h.workshopActive(c, result) {
			c.Abort()
			return
		}
		if result.MustChangePassword {
			h.fail(c, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "The password must be changed before accessing this resource.")
			c.Abort()
			return
		}
		if result.Kind == SessionKindWeb && c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead && c.Request.Method != http.MethodOptions && !h.requireCSRF(c, result) {
			c.Abort()
			return
		}
		httpx.SetPrincipal(c, result.Principal)
		c.Next()
	}
}

// restore authenticates the request through exactly one transport and
// rejects a token presented through the other one, so a mobile token cannot
// be replayed as a cookie and a browser token cannot skip CSRF as a bearer.
func (h *Handler) restore(c *gin.Context) (AuthSession, bool) {
	raw, kind, ok := h.credentials(c)
	if !ok {
		return AuthSession{}, false
	}
	result, err := h.service.Restore(c.Request.Context(), raw)
	if err != nil {
		if errors.Is(err, ErrSessionInvalid) {
			h.invalidSession(c)
		} else {
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return AuthSession{}, false
	}
	if result.Kind.normalized() != kind {
		h.invalidSession(c)
		return AuthSession{}, false
	}
	return result, true
}

func (h *Handler) credentials(c *gin.Context) (string, SessionKind, bool) {
	authorization := c.GetHeader("Authorization")
	switch {
	case h.hasCookie(c) && authorization != "":
		h.ambiguousCredentials(c)
		return "", "", false
	case authorization != "":
		if !h.mobileOriginAllowed(c) {
			h.fail(c, http.StatusForbidden, "ORIGIN_INVALID", "Request origin is not allowed.")
			return "", "", false
		}
		token, ok := bearerToken(authorization)
		if !ok {
			h.invalidSession(c)
			return "", "", false
		}
		return token, SessionKindMobile, true
	case h.hasCookie(c):
		raw, _ := c.Cookie(h.cookieName)
		return raw, SessionKindWeb, true
	default:
		h.invalidSession(c)
		return "", "", false
	}
}

func (h *Handler) hasCookie(c *gin.Context) bool {
	raw, err := c.Cookie(h.cookieName)
	return err == nil && raw != ""
}

// mobileOriginAllowed accepts the app's WebView origins and native clients
// that send no Origin header at all.
func (h *Handler) mobileOriginAllowed(c *gin.Context) bool {
	origin := strings.TrimSpace(c.GetHeader("Origin"))
	return origin == "" || security.MobileOriginAllowed(origin, h.config.MobileOrigins)
}

func (h *Handler) ambiguousCredentials(c *gin.Context) {
	h.fail(c, http.StatusBadRequest, "AUTH_AMBIGUOUS_CREDENTIALS", "Send either a session cookie or a bearer token, not both.")
}

func bearerToken(authorization string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(authorization), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" || strings.ContainsAny(token, " \t") {
		return "", false
	}
	return token, true
}

func (h *Handler) requireCSRF(c *gin.Context, session AuthSession) bool {
	if !security.ValidRequestOrigin(c.Request, h.config.AllowedOrigin) || !security.ValidCSRFToken(session.CSRFToken, c.GetHeader("X-CSRF-Token")) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.")
		return false
	}
	return true
}

func (h *Handler) workshopActive(c *gin.Context, session AuthSession) bool {
	err := h.service.EnsureWorkshopActive(c.Request.Context(), session.Principal.WorkshopID)
	switch {
	case err == nil:
		return true
	case errors.Is(err, ErrWorkshopSuspended):
		h.workshopSuspended(c)
	default:
		h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
	}
	return false
}

func (h *Handler) workshopSuspended(c *gin.Context) {
	h.fail(c, http.StatusForbidden, "WORKSHOP_SUSPENDED", "The workshop is suspended.")
}

func (h *Handler) invalidSession(c *gin.Context) {
	h.clearCookie(c)
	h.fail(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
}

func (h *Handler) setCookie(c *gin.Context, token string, expiresAt time.Time) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cookieName, Value: token, Path: "/", Secure: h.config.Secure,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, Expires: expiresAt.UTC(),
	})
}

func (h *Handler) clearCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name: h.cookieName, Value: "", Path: "/", Secure: h.config.Secure,
		HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

func (h *Handler) fail(c *gin.Context, status int, code, message string) {
	requestID := httpx.RequestIDFromContext(c)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.Header("X-Request-ID", requestID)
	c.JSON(status, gin.H{"error": gin.H{
		"code": code, "message": message, "details": gin.H{}, "request_id": requestID,
	}})
}

func sessionData(session AuthSession) gin.H {
	return gin.H{
		"expires_at":           session.ExpiresAt.UTC().Format(time.RFC3339),
		"csrf_token":           session.CSRFToken,
		"must_change_password": session.MustChangePassword,
	}
}

func requestIP(c *gin.Context) string {
	return strings.TrimSpace(c.ClientIP())
}

func decodeJSON(c *gin.Context, value any, maxBytes int64) error {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return errors.New("JSON content type required")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("multiple JSON values are not allowed")
	}
	return nil
}

func validEmail(value string) bool {
	if len(value) == 0 || len(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
