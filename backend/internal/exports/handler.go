package exports

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/clients"
	"github.com/Josed20/Tallerflow/backend/internal/orders"
	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrdersProvider interface {
	List(ctx context.Context, workshopID uuid.UUID) ([]orders.Order, error)
}

type ClientsProvider interface {
	List(ctx context.Context, workshopID uuid.UUID) ([]clients.Client, error)
}

type Handler struct {
	orders  OrdersProvider
	clients ClientsProvider
}

func NewHandler(orders OrdersProvider, clients ClientsProvider) *Handler {
	return &Handler{
		orders:  orders,
		clients: clients,
	}
}

func (h *Handler) ExportOrders(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot export data.")
		return
	}

	list, err := h.orders.List(c.Request.Context(), principal.WorkshopID)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	var buf bytes.Buffer
	// UTF-8 Byte Order Mark (BOM) para compatibilidad con Microsoft Excel
	buf.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(&buf)
	// Cabeceras en español para el taller
	header := []string{
		"Código",
		"Cliente",
		"Producto / Prenda",
		"Cantidad",
		"Unidad",
		"Estado",
		"Fecha de Promesa",
		"Fecha de Creación",
	}
	_ = w.Write(header)

	for _, o := range list {
		promised := ""
		if !o.PromisedAt.IsZero() {
			promised = o.PromisedAt.Format(time.RFC3339)
		}
		created := ""
		if !o.CreatedAt.IsZero() {
			created = o.CreatedAt.Format(time.RFC3339)
		}
		record := []string{
			o.Code,
			o.ClientName,
			o.Product,
			fmt.Sprintf("%.2f", o.Quantity),
			o.Unit,
			o.Status,
			promised,
			created,
		}
		_ = w.Write(record)
	}
	w.Flush()

	filename := fmt.Sprintf("ordenes_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func (h *Handler) ExportClients(c *gin.Context) {
	principal, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, http.StatusUnauthorized, "SESSION_INVALID", "The session is invalid or has expired.")
		return
	}

	if principal.Role != "OWNER" && principal.Role != "ADMIN" {
		httpx.RespondError(c, http.StatusForbidden, "ROLE_FORBIDDEN", "The authenticated role cannot export data.")
		return
	}

	list, err := h.clients.List(c.Request.Context(), principal.WorkshopID)
	if err != nil {
		httpx.RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred.")
		return
	}

	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")

	w := csv.NewWriter(&buf)
	header := []string{
		"Nombre / Marca",
		"Correo",
		"Teléfono",
		"Estado",
		"Fecha de Registro",
	}
	_ = w.Write(header)

	for _, cl := range list {
		email := ""
		if cl.Email != nil {
			email = *cl.Email
		}
		phone := ""
		if cl.Phone != nil {
			phone = *cl.Phone
		}
		status := "Inactivo"
		if cl.IsActive {
			status = "Activo"
		}
		created := ""
		if !cl.CreatedAt.IsZero() {
			created = cl.CreatedAt.Format(time.RFC3339)
		}
		record := []string{
			cl.Name,
			email,
			phone,
			status,
			created,
		}
		_ = w.Write(record)
	}
	w.Flush()

	filename := fmt.Sprintf("clientes_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

func RegisterRoutes(routes gin.IRouter, handler *Handler, requireSession gin.HandlerFunc) {
	routes.GET("/api/v1/exports/orders.csv", requireSession, handler.ExportOrders)
	routes.GET("/api/v1/exports/clients.csv", requireSession, handler.ExportClients)
}
