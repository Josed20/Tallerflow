package httpx

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// RouteRegistrar adds a module's routes to the shared engine.
type RouteRegistrar func(gin.IRouter)

// Dependencies contains optional infrastructure and module route hooks.
// Middleware runs for every request, including unmatched ones such as CORS
// preflights, after request IDs and panic recovery.
type Dependencies struct {
	Ping       func(context.Context) error
	Middleware []gin.HandlerFunc
	Routes     []RouteRegistrar
}

const readinessTimeout = 2 * time.Second

// NewRouter builds the shared HTTP transport and its platform endpoints.
func NewRouter(deps Dependencies) *gin.Engine {
	router := gin.New()
	router.Use(RequestID(), Recovery())
	router.Use(deps.Middleware...)

	// Uptime monitors probe with HEAD by default, and Gin answers 404 to a HEAD
	// on a GET-only route.
	healthMethods := []string{http.MethodGet, http.MethodHead}
	router.Match(healthMethods, "/health/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "alive"}})
	})
	router.Match(healthMethods, "/health/ready", func(c *gin.Context) {
		if deps.Ping == nil {
			RespondError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "A required dependency is unavailable.")
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()
		if err := deps.Ping(ctx); err != nil {
			RespondError(c, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "A required dependency is unavailable.")
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"status": "ready"}})
	})

	for _, register := range deps.Routes {
		if register != nil {
			register(router)
		}
	}

	return router
}
