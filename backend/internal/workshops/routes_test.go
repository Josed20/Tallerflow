package workshops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
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
	principal := testPrincipal()
	router := workshopRouter(t, &workshopResolverFake{}, &principal)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"data":{"userId":"`+principal.UserID.String()+`","email":"owner@example.com","displayName":"Owner","workshopId":"`+principal.WorkshopID.String()+`","role":"OWNER","passwordChangeRequired":false}}`, response.Body.String())
}

func TestMeRejectsMissingPrincipalWithAPIEnvelope(t *testing.T) {
	router := workshopRouter(t, &workshopResolverFake{}, nil)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))

	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.Contains(t, response.Body.String(), `"code":"SESSION_INVALID"`)
	require.Contains(t, response.Body.String(), response.Header().Get(httpx.RequestIDHeader))
}

func TestCurrentReturnsActiveOwnerWorkshop(t *testing.T) {
	principal := testPrincipal()
	access := Access{Workshop: Workshop{ID: principal.WorkshopID, Name: "Taller Central", Timezone: "America/Lima"}, Role: RoleOwner}
	resolver := &workshopResolverFake{access: access}
	router := workshopRouter(t, resolver, &principal)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/workshops/current", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.JSONEq(t, `{"data":{"workshop":{"id":"`+principal.WorkshopID.String()+`","name":"Taller Central","timezone":"America/Lima"},"role":"OWNER"}}`, response.Body.String())
	require.Equal(t, principal.UserID, resolver.userID)
	require.Equal(t, principal.WorkshopID, resolver.workshopID)
}

func TestCurrentRejectsMissingOrCrossTenantMembership(t *testing.T) {
	principal := testPrincipal()
	resolver := &workshopResolverFake{err: ErrMembershipNotFound}
	router := workshopRouter(t, resolver, &principal)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workshops/current?workshop_id="+uuid.NewString(), nil)
	request.Header.Set("X-Workshop-ID", uuid.NewString())
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusForbidden, response.Code)
	require.Contains(t, response.Body.String(), `"code":"WORKSHOP_FORBIDDEN"`)
	require.Equal(t, principal.WorkshopID, resolver.workshopID)
}

type workshopResolverFake struct {
	access     Access
	err        error
	userID     uuid.UUID
	workshopID uuid.UUID
}

func (r *workshopResolverFake) Resolve(_ context.Context, userID, workshopID uuid.UUID) (Access, error) {
	r.userID, r.workshopID = userID, workshopID
	return r.access, r.err
}

func workshopRouter(t *testing.T, resolver *workshopResolverFake, principal *httpx.Principal) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	handler := NewHandler(resolver)
	router := gin.New()
	router.Use(httpx.RequestID())
	requireSession := func(c *gin.Context) {
		if principal != nil {
			httpx.SetPrincipal(c, *principal)
		}
		c.Next()
	}
	RegisterRoutes(router, handler, requireSession)
	return router
}

func testPrincipal() httpx.Principal {
	return httpx.Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Role: string(RoleOwner), Email: "owner@example.com", DisplayName: "Owner"}
}
