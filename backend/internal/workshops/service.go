package workshops

import (
	"context"
	"errors"
	"fmt"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrMembershipNotFound = errors.New("active workshop membership not found")
	ErrMembershipInvalid  = errors.New("workshop membership is invalid")
)

type Workshop struct {
	ID       uuid.UUID `json:"id" gorm:"column:id"`
	Name     string    `json:"name" gorm:"column:name"`
	Timezone string    `json:"timezone" gorm:"column:timezone"`
}

func (Workshop) TableName() string { return "workshops" }

type Membership struct {
	ID         uuid.UUID `gorm:"column:id"`
	WorkshopID uuid.UUID `gorm:"column:workshop_id"`
	UserID     uuid.UUID `gorm:"column:user_id"`
	Role       Role      `gorm:"column:role"`
	Status     string    `gorm:"column:status"`
}

func (Membership) TableName() string { return "workshop_members" }

type Access struct {
	Workshop Workshop `json:"workshop"`
	Role     Role     `json:"role"`
}

type Service struct {
	tenants platformdb.TenantRunner
}

func NewService(tenants platformdb.TenantRunner) (*Service, error) {
	if tenants == nil {
		return nil, errors.New("tenant runner is required")
	}
	return &Service{tenants: tenants}, nil
}

// Resolve validates a user's active membership inside the candidate tenant.
// The workshop ID must come from trusted session state, never from a request header.
func (s *Service) Resolve(ctx context.Context, userID, workshopID uuid.UUID) (Access, error) {
	if userID == uuid.Nil || workshopID == uuid.Nil {
		return Access{}, ErrMembershipInvalid
	}

	var access Access
	err := s.tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var membership Membership
		result := tx.Where("user_id = ? AND workshop_id = ? AND status = 'ACTIVE'", userID, workshopID).Take(&membership)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrMembershipNotFound
		}
		if result.Error != nil {
			return fmt.Errorf("find membership: %w", result.Error)
		}
		if !membership.Role.Valid() {
			return ErrMembershipInvalid
		}

		var workshop Workshop
		result = tx.Where("id = ?", workshopID).Take(&workshop)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return ErrMembershipNotFound
		}
		if result.Error != nil {
			return fmt.Errorf("find workshop: %w", result.Error)
		}
		access = Access{Workshop: workshop, Role: membership.Role}
		return nil
	})
	if err != nil {
		return Access{}, err
	}
	return access, nil
}
