package workshops

import (
	"encoding/json"
	"net/http"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

// RegisterRoutes lets the central router owner compose this module.
func RegisterRoutes(mux *http.ServeMux, handler *Handler) {
	mux.HandleFunc("GET /api/v1/me", handler.Me)
	mux.HandleFunc("GET /api/v1/workshops/current", handler.Current)
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, err := httpx.PrincipalFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, principal)
}

func (h *Handler) Current(w http.ResponseWriter, r *http.Request) {
	principal, err := httpx.PrincipalFromContext(r.Context())
	if err != nil {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}
	access, err := h.service.Resolve(r.Context(), principal.UserID, principal.WorkshopID)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	writeJSON(w, http.StatusOK, access)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
