package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPrincipalRoundTripsThroughGin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	want := Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Role: "OWNER", Email: "owner@example.com", DisplayName: "Owner"}
	router := gin.New()
	router.GET("/", func(c *gin.Context) {
		SetPrincipal(c, want)
		got, err := PrincipalFromGin(c)
		require.NoError(t, err)
		require.Equal(t, want, got)
		c.Status(http.StatusNoContent)
	})
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusNoContent, response.Code)
}

func TestRequireRolesReturnsEnvelopeForMissingPrincipalAndInvalidRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		name      string
		principal *Principal
		status    int
		code      string
	}{
		{name: "missing", status: http.StatusUnauthorized, code: "SESSION_INVALID"},
		{name: "role", principal: &Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Role: "OPERATOR"}, status: http.StatusForbidden, code: "ROLE_FORBIDDEN"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestID())
			if test.principal != nil {
				router.Use(func(c *gin.Context) { SetPrincipal(c, *test.principal); c.Next() })
			}
			router.GET("/", RequireRoles("OWNER"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
			response := httptest.NewRecorder()

			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

			require.Equal(t, test.status, response.Code)
			require.NotEmpty(t, response.Header().Get(RequestIDHeader))
			var payload struct {
				Error APIError `json:"error"`
			}
			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
			require.Equal(t, test.code, payload.Error.Code)
			require.Equal(t, response.Header().Get(RequestIDHeader), payload.Error.RequestID)
		})
	}
}

func TestRequireRolesAllowsActiveOwner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	principal := Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Role: "OWNER"}
	router := gin.New()
	router.Use(func(c *gin.Context) { SetPrincipal(c, principal); c.Next() })
	router.GET("/", RequireRoles("OWNER"), func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusNoContent, response.Code)
}
