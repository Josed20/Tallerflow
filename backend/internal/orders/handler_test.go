package orders

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func TestUpdateStageRejectsSupervisor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	handler := NewHandler(NewService(repo))
	router := gin.New()
	router.PATCH("/api/v1/orders/:id/stages/:position", principalMiddleware("SUPERVISOR"), handler.UpdateStage)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/orders/"+uuid.NewString()+"/stages/1", strings.NewReader(`{"status":"COMPLETED"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("supervisor stage update = %d %s, want 403", response.Code, response.Body.String())
	}
	if repo.updateStageCalled {
		t.Fatal("forbidden stage update reached the repository")
	}
}

func TestUpdateStageRejectsStatusOtherThanCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	handler := NewHandler(NewService(repo))
	router := gin.New()
	router.PATCH("/api/v1/orders/:id/stages/:position", principalMiddleware("OPERATOR"), handler.UpdateStage)

	request := httptest.NewRequest(http.MethodPatch, "/api/v1/orders/"+uuid.NewString()+"/stages/1", strings.NewReader(`{"status":"PENDING"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("non-completion update = %d %s, want 422", response.Code, response.Body.String())
	}
	if repo.updateStageCalled {
		t.Fatal("invalid stage update reached the repository")
	}
}

func TestReopenStageRejectsOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	router := gin.New()
	router.POST("/api/v1/orders/:id/stages/:position/reopen", principalMiddleware("OPERATOR"), NewHandler(NewService(repo)).ReopenStage)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/stages/1/reopen", strings.NewReader(`{"reason":"Se marcó por error"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || repo.reopenStageCalled {
		t.Fatalf("operator correction = %d, repository called = %v", response.Code, repo.reopenStageCalled)
	}
}

func TestReopenStageRequiresReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	router := gin.New()
	router.POST("/api/v1/orders/:id/stages/:position/reopen", principalMiddleware("ADMIN"), NewHandler(NewService(repo)).ReopenStage)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/stages/1/reopen", strings.NewReader(`{"reason":"x"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnprocessableEntity || repo.reopenStageCalled {
		t.Fatalf("short reason = %d, repository called = %v", response.Code, repo.reopenStageCalled)
	}
}

func TestCancelRejectsOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	router := gin.New()
	router.POST("/api/v1/orders/:id/cancel", principalMiddleware("OPERATOR"), NewHandler(NewService(repo)).Cancel)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/cancel", strings.NewReader(`{"reason":"Pedido duplicado"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden || repo.cancelCalled {
		t.Fatalf("operator cancellation = %d, repository called = %v", response.Code, repo.cancelCalled)
	}
}

func TestUndoAllowsOperator(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &handlerRepository{}
	router := gin.New()
	router.POST("/api/v1/orders/:id/stages/:position/undo", principalMiddleware("OPERATOR"), NewHandler(NewService(repo)).UndoStage)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/orders/"+uuid.NewString()+"/stages/1/undo", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !repo.undoStageCalled {
		t.Fatalf("operator undo = %d, repository called = %v", response.Code, repo.undoStageCalled)
	}
}

func TestAddStagePhotoRejectsOversizedPayloadBeforeRepository(t *testing.T) {
	repo := &handlerRepository{}
	service := NewService(repo)

	_, err := service.AddStagePhoto(context.Background(), uuid.New(), uuid.New(), 1, "data:image/jpeg;base64,"+strings.Repeat("A", 4*1024*1024), "")

	if !errors.Is(err, ErrOrderInvalid) {
		t.Fatalf("oversized photo error = %v, want ErrOrderInvalid", err)
	}
	if repo.addStagePhotoCalled {
		t.Fatal("oversized photo reached the repository")
	}
}

func TestAddStagePhotoRejectsNonJPEGDataBeforeRepository(t *testing.T) {
	repo := &handlerRepository{}
	service := NewService(repo)

	_, err := service.AddStagePhoto(context.Background(), uuid.New(), uuid.New(), 1, "data:image/png;base64,AAAA", "")

	if !errors.Is(err, ErrOrderInvalid) {
		t.Fatalf("non-JPEG photo error = %v, want ErrOrderInvalid", err)
	}
	if repo.addStagePhotoCalled {
		t.Fatal("non-JPEG photo reached the repository")
	}
}

func principalMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		httpx.SetPrincipal(c, httpx.Principal{UserID: uuid.New(), WorkshopID: uuid.New(), Email: "worker@example.test", DisplayName: "Worker", Role: role})
		c.Next()
	}
}

type handlerRepository struct {
	updateStageCalled   bool
	reopenStageCalled   bool
	addStagePhotoCalled bool
	undoStageCalled     bool
	cancelCalled        bool
}

func (r *handlerRepository) List(context.Context, uuid.UUID) ([]Order, error) { return nil, nil }
func (r *handlerRepository) GetByID(context.Context, uuid.UUID, uuid.UUID) (*Order, error) {
	return nil, ErrOrderNotFound
}
func (r *handlerRepository) Create(context.Context, uuid.UUID, CreateOrderInput, string) (*Order, error) {
	return nil, ErrOrderInvalid
}
func (r *handlerRepository) UpdateStage(context.Context, uuid.UUID, uuid.UUID, int, uuid.UUID, string) (*OrderStage, error) {
	r.updateStageCalled = true
	return &OrderStage{}, nil
}

func (r *handlerRepository) UndoStage(context.Context, uuid.UUID, uuid.UUID, int, uuid.UUID) (*OrderStage, error) {
	r.undoStageCalled = true
	return &OrderStage{}, nil
}

func (r *handlerRepository) Cancel(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, string) (*Order, error) {
	r.cancelCalled = true
	return &Order{}, nil
}

func (r *handlerRepository) ReopenStage(context.Context, uuid.UUID, uuid.UUID, int, uuid.UUID, string) (*OrderStage, error) {
	r.reopenStageCalled = true
	return &OrderStage{}, nil
}
func (r *handlerRepository) AddStagePhoto(context.Context, uuid.UUID, uuid.UUID, int, string, string) (*StagePhoto, error) {
	r.addStagePhotoCalled = true
	return nil, ErrOrderInvalid
}
func (r *handlerRepository) GetDashboardMetrics(context.Context, uuid.UUID) (*DashboardMetrics, error) {
	return nil, nil
}
