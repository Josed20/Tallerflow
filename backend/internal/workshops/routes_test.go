package workshops

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestMeReturnsAuthenticatedPrincipal(t *testing.T) {
	principal := httpx.Principal{
		UserID: uuid.New(), WorkshopID: uuid.New(), Role: string(RoleOwner),
		Email: "owner@example.com", DisplayName: "Owner",
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	request = request.WithContext(httpx.WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()

	new(Handler).Me(response, request)
	require.Equal(t, http.StatusOK, response.Code)
	require.Contains(t, response.Body.String(), principal.UserID.String())
	require.Contains(t, response.Body.String(), `"role":"OWNER"`)
}

func TestMeRejectsMissingPrincipal(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	response := httptest.NewRecorder()
	new(Handler).Me(response, request)
	require.Equal(t, http.StatusUnauthorized, response.Code)
}
