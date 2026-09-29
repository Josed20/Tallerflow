package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHealthLiveReturnsAliveStatusAndRequestID(t *testing.T) {
	router := NewRouter(Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.NotEmpty(t, res.Header().Get("X-Request-ID"))
	require.Equal(t, "application/json; charset=utf-8", res.Header().Get("Content-Type"))
	require.JSONEq(t, `{"data":{"status":"alive"}}`, res.Body.String())
}

func TestHealthReadyFailsClosedWithoutPingDependency(t *testing.T) {
	router := NewRouter(Dependencies{})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	require.Contains(t, res.Body.String(), `"code":"DEPENDENCY_UNAVAILABLE"`)
	require.NotEmpty(t, res.Header().Get("X-Request-ID"))
}

func TestNewRouterRegistersModuleRoutes(t *testing.T) {
	router := NewRouter(Dependencies{Routes: []RouteRegistrar{
		func(routes gin.IRouter) {
			routes.GET("/api/v1/probe", func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
		},
	}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/probe", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusNoContent, res.Code)
}

func TestRecoveryDoesNotLogSecretsAndReturnsAPIError(t *testing.T) {
	var recoveryLog bytes.Buffer
	previousWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = &recoveryLog
	t.Cleanup(func() { gin.DefaultErrorWriter = previousWriter })

	router := NewRouter(Dependencies{Routes: []RouteRegistrar{
		func(routes gin.IRouter) {
			routes.GET("/api/v1/panic", func(*gin.Context) {
				panic("test panic")
			})
		},
	}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/panic", nil)
	req.Header.Set("Cookie", "tallerflow_session=cookie-secret-value")
	req.Header.Set("X-CSRF-Token", "csrf-secret-value")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusInternalServerError, res.Code)
	require.Equal(t, "application/json; charset=utf-8", res.Header().Get("Content-Type"))
	require.NotEmpty(t, res.Header().Get(RequestIDHeader))
	var body struct {
		Error APIError `json:"error"`
	}
	require.NoError(t, json.Unmarshal(res.Body.Bytes(), &body))
	require.Equal(t, "INTERNAL_ERROR", body.Error.Code)
	require.Equal(t, res.Header().Get(RequestIDHeader), body.Error.RequestID)
	require.NotContains(t, recoveryLog.String(), "cookie-secret-value")
	require.NotContains(t, recoveryLog.String(), "csrf-secret-value")
	require.NotContains(t, recoveryLog.String(), "Cookie:")
	require.NotContains(t, recoveryLog.String(), "X-Csrf-Token:")
	require.Contains(t, recoveryLog.String(), "panic recovered")
	require.Contains(t, recoveryLog.String(), res.Header().Get(RequestIDHeader))
	require.NotContains(t, recoveryLog.String(), "test panic")
}

func TestHealthReadyReturnsReadyWhenPingSucceeds(t *testing.T) {
	router := NewRouter(Dependencies{Ping: func(context.Context) error { return nil }})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusOK, res.Code)
	require.JSONEq(t, `{"data":{"status":"ready"}}`, res.Body.String())
}

func TestHealthReadyHidesPingFailure(t *testing.T) {
	router := NewRouter(Dependencies{Ping: func(context.Context) error {
		return errors.New("database-password-must-not-leak")
	}})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	require.Contains(t, res.Body.String(), `"code":"DEPENDENCY_UNAVAILABLE"`)
	require.NotContains(t, res.Body.String(), "database-password-must-not-leak")
}

func TestHealthReadyCancelsSlowPingAtDeadline(t *testing.T) {
	var observedDeadline time.Time
	router := NewRouter(Dependencies{Ping: func(ctx context.Context) error {
		observedDeadline, _ = ctx.Deadline()
		<-ctx.Done()
		return ctx.Err()
	}})
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	res := httptest.NewRecorder()
	started := time.Now()

	router.ServeHTTP(res, req)

	require.Equal(t, http.StatusServiceUnavailable, res.Code)
	require.WithinDuration(t, started.Add(2*time.Second), observedDeadline, 250*time.Millisecond)
	require.Less(t, time.Since(started), 3*time.Second)
}
