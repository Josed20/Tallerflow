package workshops

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestModelsMatchTheIdentityPersistenceContract(t *testing.T) {
	if got := (Membership{}).TableName(); got != "workshop_members" {
		t.Fatalf("membership table = %q, want workshop_members", got)
	}
	if _, ok := reflect.TypeOf(Membership{}).FieldByName("Status"); !ok {
		t.Fatal("membership model must expose the contract's status field")
	}
	if _, ok := reflect.TypeOf(Workshop{}).FieldByName("Timezone"); !ok {
		t.Fatal("workshop model must expose the contract's timezone field")
	}
}

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
