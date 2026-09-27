package onboarding

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler) {
	group := router.Group("/api/v1/onboarding")
	group.GET("/status", handler.Status)
	group.POST("/workshop", handler.Create)
}
