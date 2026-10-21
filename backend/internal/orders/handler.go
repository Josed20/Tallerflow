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
	c.Header("Cache-Control", "no-store")
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

func (h *Handler) Update(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "Only workshop managers can edit orders.")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}
	var input UpdateOrderInput
	if c.ShouldBindJSON(&input) != nil {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "ORDER_INVALID", "Review the submitted fields.")
		return
	}
	order, err := h.service.Update(c.Request.Context(), principal.WorkshopID, orderID, principal.UserID, input)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		case errors.Is(err, ErrOrderInvalid):
			httpx.RespondError(c, http.StatusUnprocessableEntity, "ORDER_INVALID", "Review the submitted fields.")
		case errors.Is(err, ErrOrderVersionConflict):
			httpx.RespondError(c, http.StatusConflict, "ORDER_VERSION_CONFLICT", "The order was edited by someone else. Reload it and try again.")
		case errors.Is(err, ErrOrderNotEditable):
			httpx.RespondError(c, http.StatusConflict, "ORDER_NOT_EDITABLE", "The order can no longer be edited.")
		default:
			httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
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

	stage, err := h.service.UpdateStage(c.Request.Context(), principal.WorkshopID, orderID, pos, principal.UserID, input.Status)
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

func (h *Handler) UndoStage(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" && principal.Role != "OPERATOR" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot undo production stages.")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}
	position, err := strconv.Atoi(c.Param("position"))
	if err != nil || position < 1 || position > 7 {
		httpx.RespondError(c, http.StatusBadRequest, "STAGE_INVALID", "The stage position must be between 1 and 7.")
		return
	}
	stage, err := h.service.UndoStage(c.Request.Context(), principal.WorkshopID, orderID, position, principal.UserID)
	if err != nil {
		switch {
		case errors.Is(err, ErrOrderNotFound):
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		case errors.Is(err, ErrStageNotFound):
			httpx.RespondError(c, http.StatusNotFound, "STAGE_NOT_FOUND", "The specified stage does not exist.")
		case errors.Is(err, ErrUndoForbidden):
			httpx.RespondError(c, http.StatusForbidden, "UNDO_FORBIDDEN", "Only the person who advanced this stage can undo it.")
		case errors.Is(err, ErrUndoWindowExpired):
			httpx.RespondError(c, http.StatusConflict, "UNDO_WINDOW_EXPIRED", "The five-minute undo window has expired.")
		case errors.Is(err, ErrNextStageHasEvidence):
			httpx.RespondError(c, http.StatusConflict, "NEXT_STAGE_HAS_EVIDENCE", "The next stage already has evidence.")
		case errors.Is(err, ErrStageNotLatest), errors.Is(err, ErrOrderInvalid):
			httpx.RespondError(c, http.StatusConflict, "STAGE_NOT_LATEST", "This is no longer the latest completed stage.")
		default:
			httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stage})
}

func (h *Handler) Cancel(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "Only workshop managers can cancel orders.")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}
	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if c.ShouldBindJSON(&input) != nil || len([]rune(strings.TrimSpace(input.Reason))) < 5 || len([]rune(input.Reason)) > 300 {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "REASON_INVALID", "A cancellation reason of 5 to 300 characters is required.")
		return
	}
	order, err := h.service.Cancel(c.Request.Context(), principal.WorkshopID, orderID, principal.UserID, input.Reason)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
			return
		}
		if errors.Is(err, ErrOrderInvalid) {
			httpx.RespondError(c, http.StatusConflict, "ORDER_NOT_CANCELLABLE", "The order can no longer be cancelled.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

func (h *Handler) ReopenStage(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "Only workshop managers can correct a completed stage.")
		return
	}
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
		return
	}
	position, err := strconv.Atoi(c.Param("position"))
	if err != nil || position < 1 || position > 7 {
		httpx.RespondError(c, http.StatusBadRequest, "STAGE_INVALID", "The stage position must be between 1 and 7.")
		return
	}
	var input struct {
		Reason string `json:"reason" binding:"required"`
	}
	if c.ShouldBindJSON(&input) != nil || len([]rune(strings.TrimSpace(input.Reason))) < 5 || len([]rune(input.Reason)) > 300 {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "REASON_INVALID", "A correction reason of 5 to 300 characters is required.")
		return
	}
	stage, err := h.service.ReopenStage(c.Request.Context(), principal.WorkshopID, orderID, position, principal.UserID, input.Reason)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
			return
		}
		if errors.Is(err, ErrStageNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "STAGE_NOT_FOUND", "The specified stage does not exist.")
			return
		}
		if errors.Is(err, ErrOrderInvalid) {
			httpx.RespondError(c, http.StatusConflict, "STAGE_NOT_REOPENABLE", "Only the last completed stage can be reopened before work starts in the next stage.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stage})
}

func (h *Handler) AddPhoto(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
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
		if errors.Is(err, ErrOrderNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "ORDER_NOT_FOUND", "The order does not exist.")
			return
		}
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
	routes.PATCH("/api/v1/orders/:id", requireSession, handler.Update)
	routes.PATCH("/api/v1/orders/:id/stages/:position", requireSession, handler.UpdateStage)
	routes.POST("/api/v1/orders/:id/stages/:position/reopen", requireSession, handler.ReopenStage)
	routes.POST("/api/v1/orders/:id/stages/:position/undo", requireSession, handler.UndoStage)
	routes.POST("/api/v1/orders/:id/cancel", requireSession, handler.Cancel)
	routes.POST("/api/v1/orders/:id/stages/:position/photos", requireSession, handler.AddPhoto)
	routes.GET("/api/v1/dashboard", requireSession, handler.Dashboard)
}
