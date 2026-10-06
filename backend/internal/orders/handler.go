package orders

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	orders, err := h.service.List(c.Request.Context(), principal.WorkshopID)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *Handler) GetByID(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}

	order, err := h.service.GetByID(c.Request.Context(), principal.WorkshopID, orderID)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": order})
}

func (h *Handler) Create(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot create orders.")
		return
	}

	idempotencyKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	var input CreateOrderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "ORDER_INVALID", "Review the submitted fields.")
		return
	}

	order, err := h.service.Create(c.Request.Context(), principal.WorkshopID, input, idempotencyKey)
	if err != nil {
		if errors.Is(err, ErrOrderInvalid) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, "ORDER_INVALID", "Review the submitted fields.")
			return
		}
		if errors.Is(err, ErrIdempotencyConflict) {
			httpx.RespondError(c, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used with a different payload.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": order})
}

func (h *Handler) UpdateStage(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" && principal.Role != "OPERATOR" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot advance production stages.")
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}

	posStr := c.Param("position")
	pos, err := strconv.Atoi(posStr)
	if err != nil || pos < 1 || pos > 7 {
		httpx.RespondError(c, http.StatusBadRequest, "STAGE_INVALID", "The stage position must be between 1 and 7.")
		return
	}

	var input UpdateStageInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "STAGE_INVALID", "Review the submitted fields.")
		return
	}
	if strings.ToUpper(strings.TrimSpace(input.Status)) != "COMPLETED" {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "STAGE_INVALID", "Only the active stage can be completed.")
		return
	}

	stage, err := h.service.UpdateStage(c.Request.Context(), principal.WorkshopID, orderID, pos, input.Status)
	if err != nil {
		if errors.Is(err, ErrStageNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "STAGE_NOT_FOUND", "The specified stage does not exist.")
			return
		}
		if errors.Is(err, ErrOrderInvalid) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, "STAGE_INVALID", "Review the submitted fields.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stage})
}

func (h *Handler) AddPhoto(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" && principal.Role != "OPERATOR" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot attach production evidence.")
		return
	}

	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}

	posStr := c.Param("position")
	pos, err := strconv.Atoi(posStr)
	if err != nil || pos < 1 || pos > 7 {
		httpx.RespondError(c, http.StatusBadRequest, "STAGE_INVALID", "The stage position must be between 1 and 7.")
		return
	}

	var input AddStagePhotoInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "PHOTO_INVALID", "Review the submitted fields.")
		return
	}

	photo, err := h.service.AddStagePhoto(c.Request.Context(), principal.WorkshopID, orderID, pos, input.PhotoData, input.Caption)
	if err != nil {
		if errors.Is(err, ErrOrderInvalid) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, "PHOTO_INVALID", "Review the submitted fields.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": photo})
}

func (h *Handler) Dashboard(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	metrics, err := h.service.GetDashboardMetrics(c.Request.Context(), principal.WorkshopID)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": metrics})
}

func RegisterRoutes(routes gin.IRouter, handler *Handler, requireSession gin.HandlerFunc) {
	routes.GET("/api/v1/orders", requireSession, handler.List)
	routes.GET("/api/v1/orders/:id", requireSession, handler.GetByID)
	routes.POST("/api/v1/orders", requireSession, handler.Create)
	routes.PATCH("/api/v1/orders/:id/stages/:position", requireSession, handler.UpdateStage)
	routes.POST("/api/v1/orders/:id/stages/:position/photos", requireSession, handler.AddPhoto)
	routes.GET("/api/v1/dashboard", requireSession, handler.Dashboard)
}
