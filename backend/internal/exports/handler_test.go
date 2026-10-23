package exports

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/clients"
	"github.com/Josed20/Tallerflow/backend/internal/orders"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockOrdersProvider struct {
	orders []orders.Order
	err    error
}

func (m *mockOrdersProvider) List(_ context.Context, _ uuid.UUID) ([]orders.Order, error) {
	return m.orders, m.err
}

type mockClientsProvider struct {
	clients []clients.Client
	err     error
}

func (m *mockClientsProvider) List(_ context.Context, _ uuid.UUID) ([]clients.Client, error) {
	return m.clients, m.err
}

func setupTestRouter(role string, ordersProv OrdersProvider, clientsProv ClientsProvider) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	handler := NewHandler(ordersProv, clientsProv)

	mockSession := func(c *gin.Context) {
		httpx.SetPrincipal(c, httpx.Principal{
			UserID:      uuid.New(),
			WorkshopID:  uuid.New(),
			Role:        role,
			Email:       "owner@tallerflowpe.com",
			DisplayName: "Lucero Demo",
		})
		c.Next()
	}

	RegisterRoutes(r, handler, mockSession)
	return r
}

func TestExportOrders_SuccessAndSanitization(t *testing.T) {
	fakeOrders := []orders.Order{
		{
			Code:       "OT-000001",
			ClientName: "=SUM(A1:A2)", // intento de fórmula
			Product:    "+Polo Malicioso", // intento de fórmula
			Quantity:   50,
			Unit:       "UNITS",
			Status:     "ACTIVE",
			PromisedAt: time.Date(2026, 10, 25, 18, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC),
		},
	}
	router := setupTestRouter("OWNER", &mockOrdersProvider{orders: fakeOrders}, &mockClientsProvider{})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/exports/orders.csv", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	body := w.Body.String()
	require.True(t, strings.HasPrefix(body, "\xEF\xBB\xBF"), "must have UTF-8 BOM")
	require.Contains(t, body, "OT-000001")
	// Verificar sanitización con comilla simple
	require.Contains(t, body, "'=SUM(A1:A2)")
	require.Contains(t, body, "'+Polo Malicioso")
	// Verificar traducción a español
	require.Contains(t, body, "En producción")
	require.Contains(t, body, "Unidades")
}

func TestExportClients_SuccessAndSanitization(t *testing.T) {
	phone := "@987654321" // empieza con @
	email := "-cliente@textil.pe" // empieza con -
	fakeClients := []clients.Client{
		{
			Name:      "Marca Gamarra",
			Email:     &email,
			Phone:     &phone,
			IsActive:  true,
			CreatedAt: time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC),
		},
	}
	router := setupTestRouter("ADMIN", &mockOrdersProvider{}, &mockClientsProvider{clients: fakeClients})

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/exports/clients.csv", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	body := w.Body.String()
	require.True(t, strings.HasPrefix(body, "\xEF\xBB\xBF"))
	require.Contains(t, body, "Marca Gamarra")
	require.Contains(t, body, "'@987654321")
	require.Contains(t, body, "'-cliente@textil.pe")
	require.Contains(t, body, "Activo")
}

func TestExport_ForbiddenForOperator(t *testing.T) {
	router := setupTestRouter("OPERATOR", &mockOrdersProvider{}, &mockClientsProvider{})

	reqOrders, _ := http.NewRequest(http.MethodGet, "/api/v1/exports/orders.csv", nil)
	wOrders := httptest.NewRecorder()
	router.ServeHTTP(wOrders, reqOrders)
	require.Equal(t, http.StatusForbidden, wOrders.Code)

	reqClients, _ := http.NewRequest(http.MethodGet, "/api/v1/exports/clients.csv", nil)
	wClients := httptest.NewRecorder()
	router.ServeHTTP(wClients, reqClients)
	require.Equal(t, http.StatusForbidden, wClients.Code)
}
