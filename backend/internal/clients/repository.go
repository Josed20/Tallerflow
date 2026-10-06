package clients

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrClientNotFound = errors.New("client not found")
	ErrClientInvalid  = errors.New("client data is invalid")
)

type Client struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	WorkshopID uuid.UUID  `json:"workshop_id" gorm:"type:uuid;not null"`
	Name       string     `json:"name" gorm:"not null"`
	Email      *string    `json:"email"`
	Phone      *string    `json:"phone"`
	IsActive   bool       `json:"is_active" gorm:"not null;default:true"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type CreateClientInput struct {
	Name  string  `json:"name"`
	Email *string `json:"email"`
	Phone *string `json:"phone"`
}

type UpdateClientInput struct {
	Name     *string `json:"name"`
	Email    *string `json:"email"`
	Phone    *string `json:"phone"`
	IsActive *bool   `json:"is_active"`
}

type Repository interface {
	List(ctx context.Context, workshopID uuid.UUID) ([]Client, error)
	GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Client, error)
	Create(ctx context.Context, workshopID uuid.UUID, input CreateClientInput) (*Client, error)
	Update(ctx context.Context, workshopID uuid.UUID, id uuid.UUID, input UpdateClientInput) (*Client, error)
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

func (r *PostgresRepository) List(ctx context.Context, workshopID uuid.UUID) ([]Client, error) {
	var clients []Client
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Raw("SELECT id, workshop_id, name, email, phone, is_active, created_at, updated_at FROM clients ORDER BY lower(name), id").Scan(&clients).Error
	})
	if err != nil {
		return nil, err
	}
	return clients, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Client, error) {
	var client Client
	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		q := tx.Raw("SELECT id, workshop_id, name, email, phone, is_active, created_at, updated_at FROM clients WHERE id = ?", id).Scan(&client)
		if q.Error != nil {
			return q.Error
		}
		if q.RowsAffected == 0 {
			return ErrClientNotFound
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (r *PostgresRepository) Create(ctx context.Context, workshopID uuid.UUID, input CreateClientInput) (*Client, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 160 {
		return nil, ErrClientInvalid
	}

	var email *string
	if input.Email != nil {
		e := strings.ToLower(strings.TrimSpace(*input.Email))
		if e != "" {
			email = &e
		}
	}

	var phone *string
	if input.Phone != nil {
		p := strings.TrimSpace(*input.Phone)
		if p != "" {
			phone = &p
		}
	}

	id := uuid.New()
	now := r.now().UTC()
	client := Client{
		ID:         id,
		WorkshopID: workshopID,
		Name:       name,
		Email:      email,
		Phone:      phone,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	err := r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Exec(
			"INSERT INTO clients (id, workshop_id, name, email, phone, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			client.ID, client.WorkshopID, client.Name, client.Email, client.Phone, client.IsActive, client.CreatedAt, client.UpdatedAt,
		).Error
	})
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func (r *PostgresRepository) Update(ctx context.Context, workshopID uuid.UUID, id uuid.UUID, input UpdateClientInput) (*Client, error) {
	existing, err := r.GetByID(ctx, workshopID, id)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		n := strings.TrimSpace(*input.Name)
		if n == "" || len(n) > 160 {
			return nil, ErrClientInvalid
		}
		existing.Name = n
	}

	if input.Email != nil {
		e := strings.ToLower(strings.TrimSpace(*input.Email))
		if e != "" {
			existing.Email = &e
		} else {
			existing.Email = nil
		}
	}

	if input.Phone != nil {
		p := strings.TrimSpace(*input.Phone)
		if p != "" {
			existing.Phone = &p
		} else {
			existing.Phone = nil
		}
	}

	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}

	existing.UpdatedAt = r.now().UTC()

	err = r.tenant.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		return tx.Exec(
			"UPDATE clients SET name = ?, email = ?, phone = ?, is_active = ?, updated_at = ? WHERE id = ?",
			existing.Name, existing.Email, existing.Phone, existing.IsActive, existing.UpdatedAt, existing.ID,
		).Error
	})
	if err != nil {
		return nil, err
	}

	return existing, nil
}
