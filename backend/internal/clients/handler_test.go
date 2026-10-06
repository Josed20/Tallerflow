package clients

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestListRejectsOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	handler := NewHandler(NewService(repo))
	router := gin.New()
	router.GET("/api/v1/clients", func(c *gin.Context) {
		httpx.SetPrincipal(c, httpx.Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Email: "operator@example.test", DisplayName: "Operator", Role: "OPERATOR"})
		c.Next()
	}, handler.List)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/clients", nil))

	if response.Code != http.StatusForbidden {
		t.Fatalf("operator client list = %d %s, want 403", response.Code, response.Body.String())
	}
	if repo.listCalled {
		t.Fatal("forbidden client list reached the repository")
	}
}

type handlerRepository struct {
	listCalled bool
}

func (r *handlerRepository) List(context.Context, uuid.UUID) ([]Client, error) {
	r.listCalled = true
	return nil, nil
}
func (r *handlerRepository) GetByID(context.Context, uuid.UUID, uuid.UUID) (*Client, error) {
	return nil, ErrClientNotFound
}
func (r *handlerRepository) Create(context.Context, uuid.UUID, CreateClientInput) (*Client, error) {
	return nil, ErrClientInvalid
}
func (r *handlerRepository) Update(context.Context, uuid.UUID, uuid.UUID, UpdateClientInput) (*Client, error) {
	return nil, ErrClientInvalid
}
