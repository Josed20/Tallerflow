package httpx

import (
	"github.com/gin-gonic/gin"
)

// APIError is the stable error envelope exposed by the HTTP API.
type APIError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details"`
	RequestID string         `json:"request_id"`
}

// RespondError writes a transport-safe error without exposing internal causes.
func RespondError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": APIError{
			Code:      code,
			Message:   message,
			Details:   map[string]any{},
			RequestID: RequestIDFromContext(c),
		},
	})
}
