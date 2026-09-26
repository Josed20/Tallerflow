package workshops

import (
	"context"
	"errors"
	"net/http"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Resolver interface {
	Resolve(context.Context, uuid.UUID, uuid.UUID) (Access, error)
}

type Handler struct {
	service Resolver
}

func NewHandler(service Resolver) *Handler { return &Handler{service: service} }

// RegisterRoutes lets the central router owner compose this module.
func RegisterRoutes(routes gin.IRouter, handler *Handler, requireSession gin.HandlerFunc) {
	routes.GET("/api/v1/me", requireSession, handler.Me)
	routes.GET("/api/v1/workshops/current", requireSession, handler.Current)
}

func (h *Handler) Me(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": principal})
}

func (h *Handler) Current(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}
	if h.service == nil {
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}
	access, err := h.service.Resolve(c.Request.Context(), principal.UserID, principal.WorkshopID)
	if err != nil {
		if errors.Is(err, ErrMembershipNotFound) || errors.Is(err, ErrMembershipInvalid) {
			httpx.RespondError(c, http.StatusForbidden, "WORKSHOP_FORBIDDEN", "The current workshop is not available.")
		} else {
			httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": access})
}
