package httpx

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRequireRoles(t *testing.T) {
	principal := Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Role: "OWNER"}
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request = request.WithContext(WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()
	RequireRoles("OWNER")(next).ServeHTTP(response, request)
	require.Equal(t, http.StatusNoContent, response.Code)

	response = httptest.NewRecorder()
	RequireRoles("ADMIN")(next).ServeHTTP(response, request)
	require.Equal(t, http.StatusForbidden, response.Code)
}
