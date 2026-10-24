package signup

import "github.com/gin-gonic/gin"

func RegisterRoutes(router gin.IRouter, handler *Handler) {
	router.POST("/api/v1/signup", handler.Register)
	verifications := router.Group("/api/v1/auth/email-verifications")
	verifications.POST("/consume", handler.Verify)
	verifications.POST("/resend", handler.Resend)
}
