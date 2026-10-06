package clients

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, workshopID uuid.UUID) ([]Client, error) {
	return s.repo.List(ctx, workshopID)
}

func (s *Service) GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Client, error) {
	return s.repo.GetByID(ctx, workshopID, id)
}

func (s *Service) Create(ctx context.Context, workshopID uuid.UUID, input CreateClientInput) (*Client, error) {
	return s.repo.Create(ctx, workshopID, input)
}

func (s *Service) Update(ctx context.Context, workshopID uuid.UUID, id uuid.UUID, input UpdateClientInput) (*Client, error) {
	return s.repo.Update(ctx, workshopID, id, input)
}
