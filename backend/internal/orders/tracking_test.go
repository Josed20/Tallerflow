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
)

type trackingRepositoryStub struct {
	createdHash []byte
	foundHash   []byte
}

func (r *trackingRepositoryStub) CreateLink(_ context.Context, _, _, _ uuid.UUID, tokenHash []byte) error {
	r.createdHash = tokenHash
	return nil
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

func TestCreateTrackingLinkStoresOnlyTheTokenHash(t *testing.T) {
	repo := &trackingRepositoryStub{}
	service := NewTrackingService(repo, bytes.NewReader(bytes.Repeat([]byte{7}, trackingTokenBytes)))

	token, err := service.CreateLink(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("create link: %v", err)
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
