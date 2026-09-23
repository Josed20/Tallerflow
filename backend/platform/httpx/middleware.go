package httpx

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

// RequestID assigns a fresh request identifier and exposes it in the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := uuid.NewString()
		c.Set(requestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)
		c.Next()
	}
}

// RequestIDFromContext returns the identifier installed by RequestID.
func RequestIDFromContext(c *gin.Context) string {
	requestID, _ := c.Get(requestIDKey)
	value, _ := requestID.(string)
	return value
}

// Recovery converts unexpected panics into the API error envelope without
// logging request headers, which can contain cookies or CSRF tokens.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				RespondError(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
			}
		}()
		c.Next()
	}
}
