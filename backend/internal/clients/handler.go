package clients

import (
	"errors"
	"net/http"

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
	if principal.Role != "OWNER" && principal.Role != "ADMIN" && principal.Role != "SUPERVISOR" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot view clients.")
		return
	}

	clients, err := h.service.List(c.Request.Context(), principal.WorkshopID)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": clients})
}

func (h *Handler) Create(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot create clients.")
		return
	}

	var input CreateClientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "CLIENT_INVALID", "Review the submitted fields.")
		return
	}

	client, err := h.service.Create(c.Request.Context(), principal.WorkshopID, input)
	if err != nil {
		if errors.Is(err, ErrClientInvalid) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, "CLIENT_INVALID", "Review the submitted fields.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": client})
}

func (h *Handler) Update(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot update clients.")
		return
	}

	clientIDStr := c.Param("id")
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		httpx.RespondError(c, http.StatusNotFound, "CLIENT_NOT_FOUND", "The client does not exist.")
		return
	}

	var input UpdateClientInput
	if err := c.ShouldBindJSON(&input); err != nil {
		httpx.RespondError(c, http.StatusUnprocessableEntity, "CLIENT_INVALID", "Review the submitted fields.")
		return
	}

	client, err := h.service.Update(c.Request.Context(), principal.WorkshopID, clientID, input)
	if err != nil {
		if errors.Is(err, ErrClientNotFound) {
			httpx.RespondError(c, http.StatusNotFound, "CLIENT_NOT_FOUND", "The client does not exist.")
			return
		}
		if errors.Is(err, ErrClientInvalid) {
			httpx.RespondError(c, http.StatusUnprocessableEntity, "CLIENT_INVALID", "Review the submitted fields.")
			return
		}
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": client})
}

func RegisterRoutes(routes gin.IRouter, handler *Handler, requireSession gin.HandlerFunc) {
	routes.GET("/api/v1/clients", requireSession, handler.List)
	routes.POST("/api/v1/clients", requireSession, handler.Create)
	routes.PATCH("/api/v1/clients/:id", requireSession, handler.Update)
}
