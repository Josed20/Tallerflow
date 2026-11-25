package orders

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductionStage struct {
	Name              string `json:"name"`
	ClientDescription string `json:"client_description"`
}
type ProductionSettings struct {
	Stages  []ProductionStage `json:"stages"`
	Version int64             `json:"version"`
}
type OrderChange struct {
	ActorName   string          `json:"actor_name"`
	ID          uuid.UUID       `json:"id"`
	ActorUserID uuid.UUID       `json:"actor_user_id"`
	Changes     json.RawMessage `json:"changes" gorm:"type:jsonb"`
	CreatedAt   time.Time       `json:"created_at"`
}
type OperationInput struct {
	ResponsibleUserID *uuid.UUID `json:"responsible_user_id"`
	BlockReason       string     `json:"block_reason"`
	Version           int64      `json:"version"`
}
type productionRepository interface {
	GetProductionSettings(context.Context, uuid.UUID) (*ProductionSettings, error)
	SaveProductionSettings(context.Context, uuid.UUID, uuid.UUID, ProductionSettings) (*ProductionSettings, error)
	UpdateOperation(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, OperationInput) (*Order, error)
}

func defaultProductionSettings() ProductionSettings {
	names := []string{"CORTE", "COSTURA", "ACABADO", "CONTROL DE CALIDAD", "PLANCHADO", "EMPAQUE", "ENTREGA"}
	descriptions := []string{"Preparamos y cortamos la tela de tus prendas.", "Confeccionamos tus prendas.", "Realizamos los acabados de tus prendas.", "Revisamos la calidad de tus prendas.", "Planchamos tus prendas.", "Empacamos tus prendas para la entrega.", "Tu pedido está listo para entregar."}
	stages := make([]ProductionStage, len(names))
	for i, name := range names {
		stages[i] = ProductionStage{name, descriptions[i]}
	}
	return ProductionSettings{Stages: stages}
}
func validateProductionSettings(input *ProductionSettings) error {
	if len(input.Stages) < 1 || len(input.Stages) > 20 || input.Version < 0 {
		return ErrOrderInvalid
	}
	seen := map[string]bool{}
	for i := range input.Stages {
		stage := &input.Stages[i]
		stage.Name = strings.TrimSpace(stage.Name)
		stage.ClientDescription = strings.TrimSpace(stage.ClientDescription)
		key := strings.ToUpper(stage.Name)
		if len([]rune(stage.Name)) < 1 || len([]rune(stage.Name)) > 80 || len([]rune(stage.ClientDescription)) < 1 || len([]rune(stage.ClientDescription)) > 500 || seen[key] {
			return ErrOrderInvalid
		}
		seen[key] = true
	}
	return nil
}
func loadProductionSettings(tx *gorm.DB, workshopID uuid.UUID) (*ProductionSettings, error) {
	var row struct {
		Stages  string
		Version int64
	}
	result := tx.Raw("SELECT stages::text AS stages, version FROM production_settings WHERE workshop_id = ?", workshopID).Scan(&row)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		defaults := defaultProductionSettings()
		return &defaults, nil
	}
	settings := &ProductionSettings{Version: row.Version}
	if err := json.Unmarshal([]byte(row.Stages), &settings.Stages); err != nil {
		return nil, err
	}
	return settings, nil
}
func (r *PostgresRepository) GetProductionSettings(ctx context.Context, workshopID uuid.UUID) (*ProductionSettings, error) {
	var settings *ProductionSettings
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var err error
		settings, err = loadProductionSettings(tx, workshopID)
		return err
	})
	return settings, err
}
func (r *PostgresRepository) SaveProductionSettings(ctx context.Context, workshopID, actorID uuid.UUID, input ProductionSettings) (*ProductionSettings, error) {
	if err := validateProductionSettings(&input); err != nil {
		return nil, err
	}
	stages, _ := json.Marshal(input.Stages)
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		result := tx.Exec(`INSERT INTO production_settings (workshop_id, stages, version) SELECT ?, ?::jsonb, 1 WHERE ? = 0
   ON CONFLICT (workshop_id) DO NOTHING`, workshopID, string(stages), input.Version)
		if result.Error != nil {
			return result.Error
		}
		if input.Version == 0 {
			if result.RowsAffected != 1 {
				return ErrOrderVersionConflict
			}
		} else {
			result = tx.Exec("UPDATE production_settings SET stages = ?::jsonb, version = version + 1, updated_at = ? WHERE workshop_id = ? AND version = ?", string(stages), r.now().UTC(), workshopID, input.Version)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrOrderVersionConflict
			}
		}
		return tx.Exec("INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, 'PRODUCTION_SETTINGS_UPDATED', jsonb_build_object('version', ?::bigint))", workshopID, actorID, input.Version+1).Error
	})
	if err != nil {
		return nil, err
	}
	input.Version++
	return &input, nil
}
func recordOrderChange(tx *gorm.DB, workshopID, orderID, actorID uuid.UUID, changes map[string]any) error {
	encoded, err := json.Marshal(changes)
	if err != nil {
		return err
	}
	return tx.Exec("INSERT INTO order_changes (workshop_id, order_id, actor_user_id, changes) VALUES (?, ?, ?, ?::jsonb)", workshopID, orderID, actorID, string(encoded)).Error
}
func (r *PostgresRepository) UpdateOperation(ctx context.Context, workshopID, orderID, actorID uuid.UUID, input OperationInput) (*Order, error) {
	input.BlockReason = strings.TrimSpace(input.BlockReason)
	if input.Version < 1 || len([]rune(input.BlockReason)) > 300 || (input.BlockReason != "" && len([]rune(input.BlockReason)) < 5) {
		return nil, ErrOrderInvalid
	}
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var current Order
		row := tx.Raw("SELECT id, status, version, responsible_user_id, block_reason FROM orders WHERE id = ? FOR UPDATE", orderID).Scan(&current)
		if row.Error != nil {
			return row.Error
		}
		if row.RowsAffected == 0 {
			return ErrOrderNotFound
		}
		if current.Status != "ACTIVE" {
			return ErrOrderNotEditable
		}
		if current.Version != input.Version {
			return ErrOrderVersionConflict
		}
		if input.ResponsibleUserID != nil {
			var count int64
			if err := tx.Raw("SELECT count(*) FROM workshop_members WHERE workshop_id = ? AND user_id = ? AND status = 'ACTIVE'", workshopID, *input.ResponsibleUserID).Scan(&count).Error; err != nil {
				return err
			}
			if count != 1 {
				return ErrOrderInvalid
			}
		}
		if err := tx.Exec("UPDATE orders SET responsible_user_id = ?, block_reason = ?, version = version + 1, updated_at = ? WHERE id = ?", input.ResponsibleUserID, input.BlockReason, r.now().UTC(), orderID).Error; err != nil {
			return err
		}
		return recordOrderChange(tx, workshopID, orderID, actorID, map[string]any{"responsible_user_id": map[string]any{"from": current.ResponsibleUserID, "to": input.ResponsibleUserID}, "block_reason": map[string]any{"from": current.BlockReason, "to": input.BlockReason}})
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, workshopID, orderID)
}
func productionError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrOrderInvalid):
		httpx.RespondError(c, 422, "PRODUCTION_INVALID", "Review the production fields.")
	case errors.Is(err, ErrOrderNotFound):
		httpx.RespondError(c, 404, "ORDER_NOT_FOUND", "The order does not exist.")
	case errors.Is(err, ErrOrderVersionConflict):
		httpx.RespondError(c, 409, "ORDER_VERSION_CONFLICT", "Reload the latest version and retry.")
	case errors.Is(err, ErrOrderNotEditable):
		httpx.RespondError(c, 409, "ORDER_NOT_EDITABLE", "The order is not active.")
	default:
		httpx.RespondError(c, 500, "INTERNAL_ERROR", "An unexpected error occurred.")
	}
}
func (h *Handler) GetProductionSettings(c *gin.Context) {
	p, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, 401, "SESSION_INVALID", "Invalid session.")
		return
	}
	repo, ok := h.service.repo.(productionRepository)
	if !ok {
		productionError(c, errors.New("repository unavailable"))
		return
	}
	settings, err := repo.GetProductionSettings(c.Request.Context(), p.WorkshopID)
	if err != nil {
		productionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": settings})
}
func (h *Handler) SaveProductionSettings(c *gin.Context) {
	p, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, 401, "SESSION_INVALID", "Invalid session.")
		return
	}
	if p.Role != "OWNER" && p.Role != "ADMIN" {
		httpx.RespondError(c, 403, "ROLE_FORBIDDEN", "Only managers can configure production.")
		return
	}
	var input ProductionSettings
	if c.ShouldBindJSON(&input) != nil {
		productionError(c, ErrOrderInvalid)
		return
	}
	repo, ok := h.service.repo.(productionRepository)
	if !ok {
		productionError(c, errors.New("repository unavailable"))
		return
	}
	result, err := repo.SaveProductionSettings(c.Request.Context(), p.WorkshopID, p.UserID, input)
	if err != nil {
		productionError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
func (h *Handler) UpdateOperation(c *gin.Context) {
	p, err := httpx.PrincipalFromGin(c)
	if err != nil {
		httpx.RespondError(c, 401, "SESSION_INVALID", "Invalid session.")
		return
	}
	if p.Role != "OWNER" && p.Role != "ADMIN" {
		httpx.RespondError(c, 403, "ROLE_FORBIDDEN", "Only managers can assign and block orders.")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		productionError(c, ErrOrderNotFound)
		return
	}
	var input OperationInput
	if c.ShouldBindJSON(&input) != nil {
		productionError(c, ErrOrderInvalid)
		return
	}
	repo, ok := h.service.repo.(productionRepository)
	if !ok {
		productionError(c, errors.New("repository unavailable"))
		return
	}
	result, err := repo.UpdateOperation(c.Request.Context(), p.WorkshopID, id, p.UserID, input)
	if err != nil {
		productionError(c, err)
		return
	}
	c.JSON(200, gin.H{"data": result})
}
