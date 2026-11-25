package exports

import (
	"context"
	"encoding/csv"
	"fmt"
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
			ClientName: "=SUM(A1:A2)",     // intento de fórmula
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
	// Tab prefix inside quoted cells prevents spreadsheet formula evaluation
	require.Contains(t, body, "\t=SUM(A1:A2)")
	require.Contains(t, body, "\t+Polo Malicioso")
	// Verificar traducción a español
	require.Contains(t, body, "En producción")
	require.Contains(t, body, "Unidades")
}

func TestExportClients_SuccessAndSanitization(t *testing.T) {
	phone := "@987654321"         // empieza con @
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
	require.Contains(t, body, "\t@987654321")
	require.Contains(t, body, "\t-cliente@textil.pe")
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

func TestSanitizeCSVFieldLocaleAndControls(t *testing.T) {
	cases := map[string]string{
		"=1+1": "\t=1+1", "+SUM(1)": "\t+SUM(1)", "-1+1": "\t-1+1", "@SUM(1)": "\t@SUM(1)",
		"＝1+1": "\t=1+1", "＋SUM(1)": "\t+SUM(1)", "－1+1": "\t-1+1", "＠SUM(1)": "\t@SUM(1)",
		"  =1": "\t=1", "\t=1": "\t=1", "\r=1": "\t=1", "\n=1": "\t=1", "\x00=1": "\t=1", "\ufeff=1": "\t=1",
		" \x01\t＝1": "\t=1", "\u0085＋1": "\t+1", "\ttext": "\ttext", "Po\x00los": "Polos", "  Taller Sur": "  Taller Sur", "": "",
	}
	for input, want := range cases {
		t.Run(fmt.Sprintf("%q", input), func(t *testing.T) { require.Equal(t, want, sanitizeCSVField(input)) })
	}
}

func readExportCSV(t *testing.T, router *gin.Engine, path string) [][]string {
	t.Helper()
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	require.Equal(t, http.StatusOK, w.Code)
	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(w.Body.String(), "\ufeff"))).ReadAll()
	require.NoError(t, err)
	return rows
}

func TestExportOrdersOperationalFieldsAndAllCellSafety(t *testing.T) {
	responsible := uuid.New()
	input := orders.Order{Code: "OT-1", ClientName: "＝1+1", Product: "Polos, \"azul\"\nL", Quantity: 600, Unit: "\x00＠SUM(1)", Status: "  ＋1", CurrentStage: "\t＝1", TrafficLight: "－1", ResponsibleUserID: &responsible, ResponsibleName: "＝1+1", BlockReason: " \r＝IMPORTDATA(1)"}
	router := setupTestRouter("ADMIN", &mockOrdersProvider{orders: []orders.Order{input}}, &mockClientsProvider{})
	rows := readExportCSV(t, router, "/api/v1/exports/orders.csv")
	require.Len(t, rows, 2)
	require.Equal(t, []string{"Código", "Cliente", "Producto / Prenda", "Cantidad", "Unidad", "Estado", "Fecha de Promesa", "Fecha de Creación", "Estación", "Semáforo", "Responsable (ID)", "Responsable", "Bloqueada", "Motivo de bloqueo"}, rows[0])
	require.Equal(t, []string{"OT-1", "\t=1+1", "Polos, \"azul\"\nL", "600.00", "\t@SUM(1)", "\t+1", "", "", "\t=1", "\t-1", responsible.String(), "\t=1+1", "Sí", "\t=IMPORTDATA(1)"}, rows[1])
}

func TestExportClientsPreservesCSVBoundaries(t *testing.T) {
	email, phone := "  ＠SUM(1)", "\r＋123"
	router := setupTestRouter("OWNER", &mockOrdersProvider{}, &mockClientsProvider{clients: []clients.Client{{Name: "＝1+2\";,＝1+2", Email: &email, Phone: &phone}}})
	rows := readExportCSV(t, router, "/api/v1/exports/clients.csv")
	require.Len(t, rows, 2)
	require.Len(t, rows[1], 5)
	require.Equal(t, "\t=1+2\";,=1+2", rows[1][0])
	require.Equal(t, "\t@SUM(1)", rows[1][1])
	require.Equal(t, "\t+123", rows[1][2])
}
