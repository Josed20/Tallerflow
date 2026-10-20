package orders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrderNotFound        = errors.New("order not found")
	ErrOrderInvalid         = errors.New("order data is invalid")
	ErrStageNotFound        = errors.New("order stage not found")
	ErrIdempotencyConflict  = errors.New("idempotency conflict")
	ErrUndoWindowExpired    = errors.New("undo window expired")
	ErrStageNotLatest       = errors.New("stage is not the latest completed stage")
	ErrNextStageHasEvidence = errors.New("next stage has evidence")
	ErrUndoForbidden        = errors.New("only the user who advanced the stage can undo it")
)

type TrafficLight string

const (
	TrafficLightOnTime    TrafficLight = "ON_TIME"
	TrafficLightAtRisk    TrafficLight = "AT_RISK"
	TrafficLightLate      TrafficLight = "LATE"
	TrafficLightCompleted TrafficLight = "COMPLETED"
)

type StagePhoto struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primaryKey"`
	WorkshopID    uuid.UUID `json:"workshop_id" gorm:"type:uuid;not null"`
	OrderID       uuid.UUID `json:"order_id" gorm:"type:uuid;not null"`
	StagePosition int       `json:"stage_position" gorm:"not null"`
	PhotoData     string    `json:"photo_data" gorm:"not null"`
	Caption       string    `json:"caption"`
	CreatedAt     time.Time `json:"created_at"`
}

type OrderStage struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	WorkshopID  uuid.UUID  `json:"workshop_id" gorm:"type:uuid;not null"`
	OrderID     uuid.UUID  `json:"order_id" gorm:"type:uuid;not null"`
	Position    int        `json:"position" gorm:"not null"`
	Name        string     `json:"name" gorm:"not null"`
	Status      string     `json:"status" gorm:"not null"` // PENDING, IN_PROGRESS, COMPLETED
	CompletedAt *time.Time `json:"completed_at"`
	CompletedBy *uuid.UUID `json:"completed_by,omitempty" gorm:"type:uuid"`
}

type Order struct {
	ID                 uuid.UUID    `json:"id" gorm:"type:uuid;primaryKey"`
	WorkshopID         uuid.UUID    `json:"workshop_id" gorm:"type:uuid;not null"`
	ClientID           uuid.UUID    `json:"client_id" gorm:"type:uuid;not null"`
	ClientName         string       `json:"client_name" gorm:"->"`
	OrderNumber        int64        `json:"order_number" gorm:"not null"`
	Code               string       `json:"code" gorm:"not null"`
	Product            string       `json:"product" gorm:"not null"`
	Specification      string       `json:"specification"`
	Quantity           float64      `json:"quantity" gorm:"not null"`
	Unit               string       `json:"unit" gorm:"not null"`
	Status             string       `json:"status" gorm:"not null;default:'ACTIVE'"` // ACTIVE, COMPLETED, CANCELLED
	PromisedAt         time.Time    `json:"promised_at" gorm:"not null"`
	Version            int64        `json:"version" gorm:"not null;default:1"`
	CreatedAt          time.Time    `json:"created_at"`
	UpdatedAt          time.Time    `json:"updated_at"`
	CancelledAt        *time.Time   `json:"cancelled_at,omitempty"`
	CancelledBy        *uuid.UUID   `json:"cancelled_by,omitempty" gorm:"type:uuid"`
	CancellationReason string       `json:"cancellation_reason,omitempty"`
	TrafficLight       TrafficLight `json:"traffic_light" gorm:"->"`
	CurrentStage       string       `json:"current_stage" gorm:"->"`
	Stages             []OrderStage `json:"stages,omitempty" gorm:"-"`
	Photos             []StagePhoto `json:"photos,omitempty" gorm:"-"`
}

type DashboardMetrics struct {
	Active    int64 `json:"active"`
	OnTime    int64 `json:"on_time"`
	AtRisk    int64 `json:"at_risk"`
	Late      int64 `json:"late"`
	Completed int64 `json:"completed"`
}

type CreateOrderInput struct {
	ClientID      uuid.UUID `json:"client_id"`
	Product       string    `json:"product"`
	Specification string    `json:"specification"`
	Quantity      float64   `json:"quantity"`
	Unit          string    `json:"unit"` // UNITS, DOZENS, KILOGRAMS, METERS
	PromisedAt    time.Time `json:"promised_at"`
}

type UpdateStageInput struct {
	Status string `json:"status"` // IN_PROGRESS, COMPLETED
}

type AddStagePhotoInput struct {
	PhotoData string `json:"photo_data"`
	Caption   string `json:"caption"`
}

type Repository interface {
	List(ctx context.Context, workshopID uuid.UUID) ([]Order, error)
	GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Order, error)
	Create(ctx context.Context, workshopID uuid.UUID, input CreateOrderInput, idempotencyKey string) (*Order, error)
	UpdateStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, status string) (*OrderStage, error)
	ReopenStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, reason string) (*OrderStage, error)
	UndoStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID) (*OrderStage, error)
	Cancel(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, actorID uuid.UUID, reason string) (*Order, error)
	AddStagePhoto(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, stagePosition int, photoData string, caption string) (*StagePhoto, error)
	GetDashboardMetrics(ctx context.Context, workshopID uuid.UUID) (*DashboardMetrics, error)
}

type PostgresRepository struct {
	tenant database.TenantRunner
	now    func() time.Time
}

func NewPostgresRepository(tenant database.TenantRunner) (*PostgresRepository, error) {
	if tenant == nil {
		return nil, errors.New("tenant runner is required")
	}
	return &PostgresRepository{tenant: tenant, now: time.Now}, nil
}

func (r *PostgresRepository) List(ctx context.Context, workshopID uuid.UUID) ([]Order, error) {
	var orders []Order
	now := r.now().UTC()
	atRiskThreshold := now.Add(48 * time.Hour)

	query := `
		SELECT
			o.id, o.workshop_id, o.client_id, c.name as client_name,
			o.order_number, o.code, o.product, o.specification,
			o.quantity, o.unit, o.status, o.promised_at, o.version,
			o.created_at, o.updated_at, o.cancelled_at, o.cancelled_by, o.cancellation_reason,
			CASE
				WHEN o.status = 'COMPLETED' THEN 'COMPLETED'
				WHEN o.promised_at < ? THEN 'LATE'
				WHEN o.promised_at < ? THEN 'AT_RISK'
				ELSE 'ON_TIME'
			END as traffic_light,
			COALESCE(
				(SELECT name FROM order_stages WHERE order_id = o.id AND status = 'IN_PROGRESS' ORDER BY position LIMIT 1),
				(SELECT name FROM order_stages WHERE order_id = o.id AND status = 'PENDING' ORDER BY position LIMIT 1),
				'ENTREGADO'
			) as current_stage
		FROM orders o
		JOIN clients c ON c.id = o.client_id
		ORDER BY
			CASE WHEN o.status = 'ACTIVE' THEN 0 ELSE 1 END,
			o.promised_at ASC,
			o.created_at DESC
	`

	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Raw(query, now, atRiskThreshold).Scan(&orders).Error
	})
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Order, error) {
	var order Order
	now := r.now().UTC()
	atRiskThreshold := now.Add(48 * time.Hour)

	query := `
		SELECT
			o.id, o.workshop_id, o.client_id, c.name as client_name,
			o.order_number, o.code, o.product, o.specification,
			o.quantity, o.unit, o.status, o.promised_at, o.version,
			o.created_at, o.updated_at, o.cancelled_at, o.cancelled_by, o.cancellation_reason,
			CASE
				WHEN o.status = 'COMPLETED' THEN 'COMPLETED'
				WHEN o.promised_at < ? THEN 'LATE'
				WHEN o.promised_at < ? THEN 'AT_RISK'
				ELSE 'ON_TIME'
			END as traffic_light
		FROM orders o
		JOIN clients c ON c.id = o.client_id
		WHERE o.id = ?
	`

	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		q := tx.Raw(query, now, atRiskThreshold, id).Scan(&order)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return ErrOrderNotFound
		}

		var stages []OrderStage
		if err := tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = ? ORDER BY position", id).Scan(&stages).Error; err != nil {
			return err
		}
		order.Stages = stages

		var photos []StagePhoto
		if err := tx.Raw("SELECT id, workshop_id, order_id, stage_position, photo_data, caption, created_at FROM stage_photos WHERE order_id = ? ORDER BY created_at ASC", id).Scan(&photos).Error; err == nil {
			order.Photos = photos
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *PostgresRepository) AddStagePhoto(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, stagePosition int, photoData string, caption string) (*StagePhoto, error) {
	if strings.TrimSpace(photoData) == "" || stagePosition < 1 || stagePosition > 7 {
		return nil, ErrOrderInvalid
	}
	now := r.now().UTC()
	photo := StagePhoto{
		ID:            uuid.New(),
		WorkshopID:    workshopID,
		OrderID:       orderID,
		StagePosition: stagePosition,
		PhotoData:     strings.TrimSpace(photoData),
		Caption:       strings.TrimSpace(caption),
		CreatedAt:     now,
	}
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Exec(
			"INSERT INTO stage_photos (id, workshop_id, order_id, stage_position, photo_data, caption, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)",
			photo.ID, photo.WorkshopID, photo.OrderID, photo.StagePosition, photo.PhotoData, photo.Caption, photo.CreatedAt,
		).Error
	})
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *PostgresRepository) Create(ctx context.Context, workshopID uuid.UUID, input CreateOrderInput, idempotencyKey string) (*Order, error) {
	if input.ClientID == uuid.Nil || strings.TrimSpace(input.Product) == "" || input.Quantity <= 0 || input.PromisedAt.IsZero() {
		return nil, ErrOrderInvalid
	}

	unit := strings.ToUpper(strings.TrimSpace(input.Unit))
	if unit != "UNITS" && unit != "DOZENS" && unit != "KILOGRAMS" && unit != "METERS" {
		unit = "UNITS"
	}

	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%g|%s|%s",
		input.ClientID, input.Product, input.Specification, input.Quantity, unit, input.PromisedAt.UTC().Format(time.RFC3339))))

	var order Order
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		// 1. Check idempotency if key provided
		if idempotencyKey != "" {
			var existing struct {
				RequestHash []byte
				ResourceID  uuid.UUID
			}
			q := tx.Raw("SELECT request_hash, resource_id FROM idempotency_keys WHERE workshop_id = ? AND key = ?", workshopID, idempotencyKey).Scan(&existing)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected == 1 {
				if hex.EncodeToString(existing.RequestHash) != hex.EncodeToString(digest[:]) {
					return ErrIdempotencyConflict
				}
				return tx.Raw("SELECT id, workshop_id, client_id, order_number, code, product, specification, quantity, unit, status, promised_at, version, created_at, updated_at FROM orders WHERE id = ?", existing.ResourceID).Scan(&order).Error
			}
		}

		// 2. Increment sequential counter per workshop
		if err := tx.Exec("INSERT INTO workshop_counters (workshop_id, next_order_number) VALUES (?, 1) ON CONFLICT (workshop_id) DO NOTHING", workshopID).Error; err != nil {
			return err
		}
		var nextNum int64
		if err := tx.Raw("UPDATE workshop_counters SET next_order_number = next_order_number + 1 WHERE workshop_id = ? RETURNING next_order_number - 1", workshopID).Scan(&nextNum).Error; err != nil {
			return err
		}

		// 3. Insert order
		orderID := uuid.New()
		code := fmt.Sprintf("OT-%06d", nextNum)
		now := r.now().UTC()
		order = Order{
			ID:            orderID,
			WorkshopID:    workshopID,
			ClientID:      input.ClientID,
			OrderNumber:   nextNum,
			Code:          code,
			Product:       strings.TrimSpace(input.Product),
			Specification: strings.TrimSpace(input.Specification),
			Quantity:      input.Quantity,
			Unit:          unit,
			Status:        "ACTIVE",
			PromisedAt:    input.PromisedAt.UTC(),
			Version:       1,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := tx.Exec(
			"INSERT INTO orders (id, workshop_id, client_id, order_number, code, product, specification, quantity, unit, status, promised_at, version, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			order.ID, order.WorkshopID, order.ClientID, order.OrderNumber, order.Code, order.Product, order.Specification, order.Quantity, order.Unit, order.Status, order.PromisedAt, order.Version, order.CreatedAt, order.UpdatedAt,
		).Error; err != nil {
			return err
		}

		// 4. Default Stages for Garment Workshop
		defaultStages := []string{
			"CORTE",
			"COSTURA",
			"ACABADO",
			"CONTROL DE CALIDAD",
			"PLANCHADO",
			"EMPAQUE",
			"ENTREGA",
		}

		for idx, stageName := range defaultStages {
			pos := idx + 1
			status := "PENDING"
			if pos == 1 {
				status = "IN_PROGRESS"
			}
			stageID := uuid.New()
			if err := tx.Exec(
				"INSERT INTO order_stages (id, workshop_id, order_id, position, name, status) VALUES (?, ?, ?, ?, ?, ?)",
				stageID, workshopID, orderID, pos, stageName, status,
			).Error; err != nil {
				return err
			}
		}

		// 5. Save Idempotency Key
		if idempotencyKey != "" {
			if err := tx.Exec(
				"INSERT INTO idempotency_keys (workshop_id, key, request_hash, resource_id, created_at) VALUES (?, ?, ?, ?, ?)",
				workshopID, idempotencyKey, digest[:], orderID, now,
			).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, workshopID, order.ID)
}

func (r *PostgresRepository) UpdateStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, status string) (*OrderStage, error) {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != "COMPLETED" {
		return nil, ErrOrderInvalid
	}

	var updatedStage OrderStage
	now := r.now().UTC()

	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var orderStatus string
		orderRow := tx.Raw("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderID).Scan(&orderStatus)
		if orderRow.Error != nil {
			return orderRow.Error
		}
		if orderRow.RowsAffected == 0 {
			return ErrOrderNotFound
		}
		if orderStatus != "ACTIVE" {
			return ErrOrderInvalid
		}
		var current OrderStage
		q := tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = ? AND position = ? FOR UPDATE", orderID, position).Scan(&current)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return ErrStageNotFound
		}

		result := tx.Exec(
			"UPDATE order_stages SET status = ?, completed_at = ?, completed_by = ? WHERE id = ? AND status = 'IN_PROGRESS'",
			status, now, actorID, current.ID,
		)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOrderInvalid
		}

		if position < 7 {
			next := tx.Exec(
				"UPDATE order_stages SET status = 'IN_PROGRESS' WHERE order_id = ? AND position = ? AND status = 'PENDING'",
				orderID, position+1,
			)
			if next.Error != nil {
				return next.Error
			}
			if next.RowsAffected != 1 {
				return ErrOrderInvalid
			}
		}

		if position == 7 {
			result = tx.Exec("UPDATE orders SET status = 'COMPLETED', version = version + 1, updated_at = ? WHERE id = ? AND status = 'ACTIVE'", now, orderID)
		} else {
			result = tx.Exec("UPDATE orders SET version = version + 1, updated_at = ? WHERE id = ? AND status = 'ACTIVE'", now, orderID)
		}
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOrderInvalid
		}

		return tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE id = ?", current.ID).Scan(&updatedStage).Error
	})
	if err != nil {
		return nil, err
	}

	return &updatedStage, nil
}

func (r *PostgresRepository) ReopenStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, reason string) (*OrderStage, error) {
	if position < 1 || position > 7 || len([]rune(strings.TrimSpace(reason))) < 5 || len([]rune(reason)) > 300 {
		return nil, ErrOrderInvalid
	}
	var reopened OrderStage
	now := r.now().UTC()
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var orderStatus string
		orderRow := tx.Raw("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderID).Scan(&orderStatus)
		if orderRow.Error != nil {
			return orderRow.Error
		}
		if orderRow.RowsAffected == 0 {
			return ErrOrderNotFound
		}
		if (position == 7 && orderStatus != "COMPLETED") || (position < 7 && orderStatus != "ACTIVE") {
			return ErrOrderInvalid
		}

		stageRow := tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = ? AND position = ? FOR UPDATE", orderID, position).Scan(&reopened)
		if stageRow.Error != nil {
			return stageRow.Error
		}
		if stageRow.RowsAffected == 0 {
			return ErrStageNotFound
		}
		if reopened.Status != "COMPLETED" {
			return ErrOrderInvalid
		}

		if position < 7 {
			var next OrderStage
			nextRow := tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = ? AND position = ? FOR UPDATE", orderID, position+1).Scan(&next)
			if nextRow.Error != nil {
				return nextRow.Error
			}
			if nextRow.RowsAffected == 0 || next.Status != "IN_PROGRESS" {
				return ErrOrderInvalid
			}
			var photoCount int64
			if err := tx.Raw("SELECT count(*) FROM stage_photos WHERE order_id = ? AND stage_position = ?", orderID, position+1).Scan(&photoCount).Error; err != nil {
				return err
			}
			if photoCount > 0 {
				return ErrNextStageHasEvidence
			}
			resetNext := tx.Exec("UPDATE order_stages SET status = 'PENDING' WHERE id = ? AND status = 'IN_PROGRESS'", next.ID)
			if resetNext.Error != nil {
				return resetNext.Error
			}
			if resetNext.RowsAffected != 1 {
				return ErrOrderInvalid
			}
		}

		resetStage := tx.Exec("UPDATE order_stages SET status = 'IN_PROGRESS', completed_at = NULL, completed_by = NULL WHERE id = ? AND status = 'COMPLETED'", reopened.ID)
		if resetStage.Error != nil {
			return resetStage.Error
		}
		if resetStage.RowsAffected != 1 {
			return ErrOrderInvalid
		}
		updatedOrder := tx.Exec("UPDATE orders SET status = 'ACTIVE', version = version + 1, updated_at = ? WHERE id = ?", now, orderID)
		if updatedOrder.Error != nil {
			return updatedOrder.Error
		}
		if updatedOrder.RowsAffected != 1 {
			return ErrOrderInvalid
		}
		audit := tx.Exec("INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, 'ORDER_STAGE_REOPENED', jsonb_build_object('order_id', ?::text, 'stage_position', ?::integer, 'reason', ?::text))", workshopID, actorID, orderID.String(), position, strings.TrimSpace(reason))
		if audit.Error != nil {
			return audit.Error
		}
		return tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE id = ?", reopened.ID).Scan(&reopened).Error
	})
	if err != nil {
		return nil, err
	}
	return &reopened, nil
}

func (r *PostgresRepository) UndoStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID) (*OrderStage, error) {
	if position < 1 || position > 7 || actorID == uuid.Nil {
		return nil, ErrOrderInvalid
	}
	var reopened OrderStage
	now := r.now().UTC()
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var orderStatus string
		orderRow := tx.Raw("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderID).Scan(&orderStatus)
		if orderRow.Error != nil {
			return orderRow.Error
		}
		if orderRow.RowsAffected == 0 {
			return ErrOrderNotFound
		}
		if (position == 7 && orderStatus != "COMPLETED") || (position < 7 && orderStatus != "ACTIVE") {
			return ErrStageNotLatest
		}

		stageRow := tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = ? AND position = ? FOR UPDATE", orderID, position).Scan(&reopened)
		if stageRow.Error != nil {
			return stageRow.Error
		}
		if stageRow.RowsAffected == 0 {
			return ErrStageNotFound
		}
		if reopened.Status != "COMPLETED" || reopened.CompletedAt == nil {
			return ErrStageNotLatest
		}
		if reopened.CompletedBy == nil || *reopened.CompletedBy != actorID {
			return ErrUndoForbidden
		}
		if now.Sub(reopened.CompletedAt.UTC()) > 5*time.Minute {
			return ErrUndoWindowExpired
		}

		if position < 7 {
			var next OrderStage
			nextRow := tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE order_id = ? AND position = ? FOR UPDATE", orderID, position+1).Scan(&next)
			if nextRow.Error != nil {
				return nextRow.Error
			}
			if nextRow.RowsAffected == 0 || next.Status != "IN_PROGRESS" {
				return ErrStageNotLatest
			}
			var photoCount int64
			if err := tx.Raw("SELECT count(*) FROM stage_photos WHERE order_id = ? AND stage_position = ?", orderID, position+1).Scan(&photoCount).Error; err != nil {
				return err
			}
			if photoCount > 0 {
				return ErrNextStageHasEvidence
			}
			result := tx.Exec("UPDATE order_stages SET status = 'PENDING' WHERE id = ? AND status = 'IN_PROGRESS'", next.ID)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return ErrStageNotLatest
			}
		}

		result := tx.Exec("UPDATE order_stages SET status = 'IN_PROGRESS', completed_at = NULL, completed_by = NULL WHERE id = ? AND status = 'COMPLETED'", reopened.ID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrStageNotLatest
		}
		result = tx.Exec("UPDATE orders SET status = 'ACTIVE', version = version + 1, updated_at = ? WHERE id = ?", now, orderID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOrderInvalid
		}
		if err := tx.Exec("INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, 'ORDER_STAGE_UNDONE', jsonb_build_object('order_id', ?::text, 'stage_position', ?::integer, 'client', 'web'))", workshopID, actorID, orderID.String(), position).Error; err != nil {
			return err
		}
		return tx.Raw("SELECT id, workshop_id, order_id, position, name, status, completed_at, completed_by FROM order_stages WHERE id = ?", reopened.ID).Scan(&reopened).Error
	})
	if err != nil {
		return nil, err
	}
	return &reopened, nil
}

func (r *PostgresRepository) Cancel(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, actorID uuid.UUID, reason string) (*Order, error) {
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) < 5 || len([]rune(reason)) > 300 {
		return nil, ErrOrderInvalid
	}
	now := r.now().UTC()
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var status string
		row := tx.Raw("SELECT status FROM orders WHERE id = ? FOR UPDATE", orderID).Scan(&status)
		if row.Error != nil {
			return row.Error
		}
		if row.RowsAffected == 0 {
			return ErrOrderNotFound
		}
		if status == "CANCELLED" {
			return ErrOrderInvalid
		}
		result := tx.Exec("UPDATE orders SET status = 'CANCELLED', cancelled_at = ?, cancelled_by = ?, cancellation_reason = ?, version = version + 1, updated_at = ? WHERE id = ? AND status <> 'CANCELLED'", now, actorID, reason, now, orderID)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrOrderInvalid
		}
		if err := tx.Exec("INSERT INTO audit_events (workshop_id, actor_user_id, event_type, details) VALUES (?, ?, 'ORDER_CANCELLED', jsonb_build_object('order_id', ?::text, 'reason', ?::text))", workshopID, actorID, orderID.String(), reason).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, workshopID, orderID)
}

func (r *PostgresRepository) GetDashboardMetrics(ctx context.Context, workshopID uuid.UUID) (*DashboardMetrics, error) {
	var metrics DashboardMetrics
	now := r.now().UTC()
	atRiskThreshold := now.Add(48 * time.Hour)

	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'ACTIVE') as active,
			COUNT(*) FILTER (WHERE status = 'ACTIVE' AND promised_at >= ?) as on_time,
			COUNT(*) FILTER (WHERE status = 'ACTIVE' AND promised_at >= ? AND promised_at < ?) as at_risk,
			COUNT(*) FILTER (WHERE status = 'ACTIVE' AND promised_at < ?) as late,
			COUNT(*) FILTER (WHERE status = 'COMPLETED') as completed
		FROM orders
	`

	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Raw(query, atRiskThreshold, now, atRiskThreshold, now).Scan(&metrics).Error
	})
	if err != nil {
		return nil, err
	}

	return &metrics, nil
}
