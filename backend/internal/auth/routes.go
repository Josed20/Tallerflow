package auth

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler) {
	auth := router.Group("/api/v1/auth")
	auth.POST("/login", handler.Login)
	auth.POST("/logout", handler.Logout)
	auth.GET("/session", handler.Session)
	auth.POST("/change-password", handler.ChangePassword)
}
