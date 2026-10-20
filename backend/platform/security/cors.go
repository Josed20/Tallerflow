package security

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// MobileOriginAllowed reports whether origin is exactly one of the app's
// WebView origins. Comparison ignores case only; ports, paths and look-alike
// hosts never match.
func MobileOriginAllowed(origin string, allowed []string) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" || origin == "null" {
		return false
	}
	for _, candidate := range allowed {
		if strings.EqualFold(origin, candidate) {
			return true
		}
	}
	return false
}

// MobileCORS lets the Android app, whose WebView runs on https://localhost,
// call the API on another origin. Only exact listed origins receive CORS
// headers, and never Access-Control-Allow-Credentials: the app authenticates
// with a bearer token, so the browser cookie session stays same-origin only.
// Requests from other origins pass through untouched, which keeps the web
// app's same-origin flow unchanged; their preflights are refused.
func MobileCORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		preflight := c.Request.Method == http.MethodOptions && c.GetHeader("Access-Control-Request-Method") != ""
		if !MobileOriginAllowed(origin, allowedOrigins) {
			if preflight && origin != "" {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.Next()
			return
		}

		header := c.Writer.Header()
		header.Set("Access-Control-Allow-Origin", origin)
		header.Add("Vary", "Origin")
		header.Set("Access-Control-Expose-Headers", "X-Request-ID, Retry-After")
		if preflight {
			header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key")
			header.Set("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
