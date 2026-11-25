package orders

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type trackingRepositoryStub struct {
	createdHash []byte
	action      string
	foundHash   []byte
}

func (r *trackingRepositoryStub) ManageLink(_ context.Context, _, _, _ uuid.UUID, action string) (string, error) {
	r.action = action
	token := strings.Repeat("A", 43)
	r.createdHash = trackingDigest(token)
	return token, nil
}

func (r *trackingRepositoryStub) FindByToken(_ context.Context, tokenHash []byte) (*TrackingOrder, error) {
	r.foundHash = tokenHash
	return &TrackingOrder{Code: "OT-000001"}, nil
}

func trackingRouter(repo TrackingRepository, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterTrackingRoutes(router, NewTrackingHandler(NewTrackingService(repo, nil)), principalMiddleware(role))
	return router
}

func TestCreateTrackingLinkRejectsOperator(t *testing.T) {
	repo := &trackingRepositoryStub{}
	response := httptest.NewRecorder()
	trackingRouter(repo, "OPERATOR").ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/tracking-link", nil))

	if response.Code != http.StatusForbidden {
		t.Fatalf("operator tracking link = %d %s, want 403", response.Code, response.Body.String())
	}
	if repo.createdHash != nil {
		t.Fatal("forbidden tracking link reached the repository")
	}
}

func TestCreateTrackingLinkUsesRecovery(t *testing.T) {
	repo := &trackingRepositoryStub{}
	service := NewTrackingService(repo, nil)

	token, err := service.CreateLink(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("create link: %v", err)
	}
	if repo.action != "recover" {
		t.Fatalf("action = %q", repo.action)
	}
	if !trackingTokenFormat.MatchString(token) {
		t.Fatalf("token %q does not match the public format", token)
	}
	if bytes.Equal(repo.createdHash, []byte(token)) || len(repo.createdHash) != 32 {
		t.Fatalf("repository received %d bytes, want a 32-byte digest of the token", len(repo.createdHash))
	}
	if !bytes.Equal(repo.createdHash, trackingDigest(token)) {
		t.Fatal("stored hash does not match the token digest")
	}
}

func TestTrackingLookupRejectsMalformedTokenWithoutQuery(t *testing.T) {
	repo := &trackingRepositoryStub{}
	response := httptest.NewRecorder()
	trackingRouter(repo, "OWNER").ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tracking/"+strings.Repeat("x", 12), nil))

	if response.Code != http.StatusNotFound {
		t.Fatalf("malformed token = %d, want 404", response.Code)
	}
	if repo.foundHash != nil {
		t.Fatal("malformed token reached the repository")
	}
}

func TestTrackingLookupIsPublicAndNotCached(t *testing.T) {
	repo := &trackingRepositoryStub{}
	gin.SetMode(gin.TestMode)
	router := gin.New()
	denySession := func(c *gin.Context) { c.AbortWithStatus(http.StatusUnauthorized) }
	RegisterTrackingRoutes(router, NewTrackingHandler(NewTrackingService(repo, nil)), denySession)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/tracking/"+strings.Repeat("A", 43), nil))

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "OT-000001") {
		t.Fatalf("public lookup = %d %s, want 200 with the order", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestTrackingManagementPermissionsAndActions(t *testing.T) {
	for _, role := range []string{"OPERATOR", "SUPERVISOR", "OWNER", "ADMIN"} {
		for _, action := range []string{"replace", "revoke"} {
			repo := &trackingRepositoryStub{}
			response := httptest.NewRecorder()
			trackingRouter(repo, role).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/tracking-link/"+action, nil))
			if role == "OWNER" || role == "ADMIN" {
				if response.Code != http.StatusCreated || repo.action != action {
					t.Fatalf("%s %s: %d action %q", role, action, response.Code, repo.action)
				}
			} else if response.Code != http.StatusForbidden || repo.action != "" {
				t.Fatalf("%s %s reached management", role, action)
			}
		}
	}
	response := httptest.NewRecorder()
	repo := &trackingRepositoryStub{}
	trackingRouter(repo, "OWNER").ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/tracking-link/rotate", nil))
	if response.Code != http.StatusBadRequest || repo.action != "" {
		t.Fatal("unknown action must not mutate links")
	}
}

func TestTrackingRecoveryRequiresSecretAndBindsTenantOrderNonce(t *testing.T) {
	secret := []byte("a server-only secret")
	w, orderID := uuid.New(), uuid.New()
	token := recoverTrackingToken(secret, w, orderID, "nonce")
	if token != recoverTrackingToken(secret, w, orderID, "nonce") {
		t.Fatal("recovery changed the token")
	}
	for _, other := range []string{
		recoverTrackingToken([]byte("different"), w, orderID, "nonce"),
		recoverTrackingToken(secret, uuid.New(), orderID, "nonce"),
		recoverTrackingToken(secret, w, uuid.New(), "nonce"),
		recoverTrackingToken(secret, w, orderID, "different"),
	} {
		if token == other {
			t.Fatal("tracking token not bound to recovery inputs")
		}
	}
	r := &PostgresTrackingRepository{}
	if _, err := r.ManageLink(context.Background(), w, orderID, uuid.New(), "recover"); err == nil {
		t.Fatal("missing secret must fail before querying")
	}
}

func TestTrackingRepositoryUsesInjectedSecretInsteadOfEnvironment(t *testing.T) {
	fixture, _ := newOrderRepositoryFixture(t)
	const configured = "configured-tracking-secret-for-the-application"
	for _, environment := range []string{"", "a-different-process-environment-secret"} {
		t.Setenv("TF_SESSION_PEPPER", environment)
		repo, err := NewPostgresTrackingRepository(&gorm.DB{}, fixture.tenant, configured)
		if err != nil {
			t.Fatalf("construct with injected configuration: %v", err)
		}
		if !bytes.Equal(repo.secret, []byte(configured)) {
			t.Fatal("tracking did not use the injected application secret")
		}
	}
}

func TestTrackingRepositoryRejectsMissingDependenciesAndSecretAtConstruction(t *testing.T) {
	fixture, _ := newOrderRepositoryFixture(t)
	t.Setenv("TF_SESSION_PEPPER", "environment-must-not-rescue-empty-configuration")
	for _, secret := range []string{"", " \t\n"} {
		if _, err := NewPostgresTrackingRepository(&gorm.DB{}, fixture.tenant, secret); err == nil {
			t.Fatal("missing configured secret must fail during construction")
		}
	}
	if _, err := NewPostgresTrackingRepository(nil, fixture.tenant, "configured-secret"); err == nil {
		t.Fatal("missing database must fail during construction")
	}
	if _, err := NewPostgresTrackingRepository(&gorm.DB{}, nil, "configured-secret"); err == nil {
		t.Fatal("missing tenant runner must fail during construction")
	}
}
