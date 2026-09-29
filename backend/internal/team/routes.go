package team

import "github.com/gin-gonic/gin"

func RegisterRoutes(routes gin.IRouter, handler *Handler, requireSession gin.HandlerFunc) {
	team := routes.Group("/api/v1/team", requireSession)
	team.GET("", handler.List)
	team.POST("/invitations", handler.Invite)
	team.PATCH("/:membershipId", handler.UpdateMember)

	routes.POST("/api/v1/team/invitations/consume", handler.Consume)
}
