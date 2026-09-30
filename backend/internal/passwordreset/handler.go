package passwordreset

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UseCases interface {
	RequestReset(ctx gin.Context, email, ip string)
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type RequestResetInput struct {
	Email string `json:"email"`
}

type ConsumeResetInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (h *Handler) RequestReset(c *gin.Context) {
	var input RequestResetInput
	if err := decodeJSON(c, &input, 4096); err != nil || strings.TrimSpace(input.Email) == "" {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "El correo proporcionado es inválido.")
		return
	}

	ip := strings.TrimSpace(c.ClientIP())
	err := h.service.RequestReset(c.Request.Context(), input.Email, ip)
	if err != nil {
		switch {
		case errors.Is(err, ErrEmailInvalid):
			h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "El correo proporcionado es inválido.")
		case errors.Is(err, ErrRateLimited):
			c.Header("Retry-After", "900")
			h.fail(c, http.StatusTooManyRequests, "RATE_LIMITED", "Demasiados intentos. Inténtalo más tarde.")
		default:
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "No se pudo procesar la solicitud.")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":  "REQUESTED",
			"message": "Si el correo está registrado, recibirás un enlace para restablecer tu contraseña.",
		},
	})
}

func (h *Handler) ConsumeReset(c *gin.Context) {
	var input ConsumeResetInput
	if err := decodeJSON(c, &input, 1<<20+4096); err != nil {
		h.fail(c, http.StatusBadRequest, "INVALID_REQUEST", "La solicitud es inválida.")
		return
	}

	if strings.TrimSpace(input.Token) == "" {
		h.fail(c, http.StatusBadRequest, "TOKEN_INVALID", "El token de recuperación es requerido.")
		return
	}

	if len(input.NewPassword) < 12 {
		h.fail(c, http.StatusBadRequest, "PASSWORD_TOO_WEAK", "La nueva contraseña debe tener al menos 12 caracteres.")
		return
	}

	err := h.service.ConsumeReset(c.Request.Context(), input.Token, input.NewPassword, strings.TrimSpace(c.ClientIP()))
	if err != nil {
		switch {
		case errors.Is(err, ErrTokenInvalid):
			h.fail(c, http.StatusBadRequest, "TOKEN_INVALID", "El enlace de recuperación es inválido.")
		case errors.Is(err, ErrTokenExpired):
			h.fail(c, http.StatusBadRequest, "TOKEN_EXPIRED", "El enlace de recuperación ha expirado.")
		case errors.Is(err, ErrTokenAlreadyUsed):
			h.fail(c, http.StatusBadRequest, "TOKEN_ALREADY_USED", "El enlace de recuperación ya ha sido utilizado.")
		case errors.Is(err, ErrPasswordTooWeak):
			h.fail(c, http.StatusBadRequest, "PASSWORD_TOO_WEAK", "La nueva contraseña debe tener al menos 12 caracteres.")
		case errors.Is(err, ErrRateLimited):
			c.Header("Retry-After", "900")
			h.fail(c, http.StatusTooManyRequests, "RATE_LIMITED", "Demasiados intentos. Inténtalo más tarde.")
		default:
			h.fail(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Ocurrió un error inesperado al actualizar la contraseña.")
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":  "PASSWORD_RESET_COMPLETED",
			"message": "Tu contraseña ha sido restablecida exitosamente.",
		},
	})
}

func (h *Handler) fail(c *gin.Context, status int, code, message string) {
	requestID := httpx.RequestIDFromContext(c)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	c.Header("X-Request-ID", requestID)
	c.JSON(status, gin.H{
		"error": gin.H{
			"code":       code,
			"message":    message,
			"details":    gin.H{},
			"request_id": requestID,
		},
	})
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
