package passwordreset

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(routes gin.IRouter, handler *Handler) {
	routes.POST("/api/v1/auth/password-resets", handler.RequestReset)
	routes.POST("/api/v1/auth/password-resets/consume", handler.ConsumeReset)
}
