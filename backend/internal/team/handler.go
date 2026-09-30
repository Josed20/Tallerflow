package team

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service        *Service
	allowedOrigin  string
	consumeLimiter *security.MemoryRateLimiter
}

func NewHandler(service *Service, allowedOrigin string, rateLimitSecret ...[]byte) (*Handler, error) {
	if service == nil {
		return nil, errors.New("team service is required")
	}
	if strings.TrimSpace(allowedOrigin) == "" {
		return nil, errors.New("allowed origin is required")
	}
	secret := []byte("team-invitation-consume-rate-limit")
	if len(rateLimitSecret) > 0 && len(rateLimitSecret[0]) > 0 {
		secret = rateLimitSecret[0]
	}
	return &Handler{
		service: service, allowedOrigin: allowedOrigin,
		consumeLimiter: security.NewMemoryRateLimiter(secret, time.Now, 15*time.Minute, 5, 10000),
	}, nil
}

type inviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

type consumeRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type updateMemberRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

func (h *Handler) List(c *gin.Context) {
	principal, ok := h.principal(c)
	if !ok {
		return
	}
	members, invitations, err := h.service.List(c.Request.Context(), principal)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if members == nil {
		members = []Member{}
	}
	if invitations == nil {
		invitations = []Invitation{}
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"members": members, "invitations": invitations}})
}

func (h *Handler) Invite(c *gin.Context) {
	principal, ok := h.principal(c)
	if !ok {
		return
	}
	if !security.ValidRequestOrigin(c.Request, h.allowedOrigin) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.")
		return
	}
	var request inviteRequest
	if err := decodeJSON(c, &request, 4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
		return
	}
	created, err := h.service.Invite(c.Request.Context(), principal, InviteInput{Email: request.Email, Role: request.Role})
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": created})
}

func (h *Handler) RegenerateInvitation(c *gin.Context) {
	principal, ok := h.principal(c)
	if !ok {
		return
	}
	if !security.ValidRequestOrigin(c.Request, h.allowedOrigin) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.")
		return
	}
	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		h.fail(c, http.StatusNotFound, "TEAM_INVITATION_UNAVAILABLE", "Invitation is unavailable.")
		return
	}
	created, err := h.service.RegenerateInvitation(c.Request.Context(), principal, invitationID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": created})
}

func (h *Handler) CancelInvitation(c *gin.Context) {
	principal, ok := h.principal(c)
	if !ok {
		return
	}
	if !security.ValidRequestOrigin(c.Request, h.allowedOrigin) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.")
		return
	}
	invitationID, err := uuid.Parse(c.Param("invitationId"))
	if err != nil {
		h.fail(c, http.StatusNotFound, "TEAM_INVITATION_UNAVAILABLE", "Invitation is unavailable.")
		return
	}
	if err := h.service.CancelInvitation(c.Request.Context(), principal, invitationID); err != nil {
		h.respondError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Consume(c *gin.Context) {
	if !security.ValidRequestOrigin(c.Request, h.allowedOrigin) {
		h.fail(c, http.StatusForbidden, "ORIGIN_INVALID", "Request origin is not allowed.")
		return
	}
	var request consumeRequest
	if err := decodeJSON(c, &request, 1<<20+4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
		return
	}
	if !h.consumeLimiter.Allow(c.ClientIP()) {
		c.Header("Retry-After", "900")
		h.fail(c, http.StatusTooManyRequests, "RATE_LIMITED", "Demasiados intentos. Inténtalo más tarde.")
		return
	}
	result, err := h.service.Consume(c.Request.Context(), ConsumeInput{
		Token: request.Token, Name: request.Name, Password: request.Password,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) UpdateMember(c *gin.Context) {
	principal, ok := h.principal(c)
	if !ok {
		return
	}
	if !security.ValidRequestOrigin(c.Request, h.allowedOrigin) {
		h.fail(c, http.StatusForbidden, "CSRF_INVALID", "CSRF validation failed.")
		return
	}
	membershipID, err := uuid.Parse(c.Param("membershipId"))
	if err != nil {
		h.fail(c, http.StatusNotFound, "TEAM_MEMBER_NOT_FOUND", "Team member was not found.")
		return
	}
	var request updateMemberRequest
	if err := decodeJSON(c, &request, 4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "The request is invalid.")
		return
	}
	member, err := h.service.UpdateMember(c.Request.Context(), principal, membershipID, UpdateMemberInput{
		Role: request.Role, Status: request.Status,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": member})
}

func (h *Handler) principal(c *gin.Context) (httpx.Principal, bool) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		h.fail(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return httpx.Principal{}, false
	}
	return principal, true
}

func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		h.fail(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot access this resource.")
	case errors.Is(err, ErrInvalidInput):
		h.fail(c, http.StatusUnprocessableEntity, "TEAM_INVALID", "Team input is invalid.")
	case errors.Is(err, ErrDuplicateInvitation):
		h.fail(c, http.StatusConflict, "TEAM_INVITATION_DUPLICATE", "An active invitation already exists.")
	case errors.Is(err, ErrInvitationUnavailable), errors.Is(err, ErrCrossWorkshopUser):
		h.fail(c, http.StatusUnprocessableEntity, "TEAM_INVITATION_UNAVAILABLE", "Invitation is unavailable.")
	case errors.Is(err, ErrLastOwner):
		h.fail(c, http.StatusConflict, "TEAM_LAST_OWNER", "The workshop must keep one active owner.")
	case errors.Is(err, ErrMemberNotFound):
		h.fail(c, http.StatusNotFound, "TEAM_MEMBER_NOT_FOUND", "Team member was not found.")
	default:
		h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
	}
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
