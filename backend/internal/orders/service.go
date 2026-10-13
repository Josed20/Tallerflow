package orders

import (
	"bytes"
	"context"
	"encoding/base64"
	"image/jpeg"
	"strings"

	"github.com/google/uuid"
)

const maxStagePhotoBytes = 3 * 1024 * 1024

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context, workshopID uuid.UUID) ([]Order, error) {
	return s.repo.List(ctx, workshopID)
}

func (s *Service) GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Order, error) {
	return s.repo.GetByID(ctx, workshopID, id)
}

func (s *Service) Create(ctx context.Context, workshopID uuid.UUID, input CreateOrderInput, idempotencyKey string) (*Order, error) {
	return s.repo.Create(ctx, workshopID, input, idempotencyKey)
}

func (s *Service) UpdateStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, status string) (*OrderStage, error) {
	return s.repo.UpdateStage(ctx, workshopID, orderID, position, status)
}

func (s *Service) ReopenStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, reason string) (*OrderStage, error) {
	return s.repo.ReopenStage(ctx, workshopID, orderID, position, actorID, reason)
}

func (s *Service) AddStagePhoto(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, photoData string, caption string) (*StagePhoto, error) {
	if !validStagePhoto(photoData) || len([]rune(strings.TrimSpace(caption))) > 300 {
		return nil, ErrOrderInvalid
	}
	return s.repo.AddStagePhoto(ctx, workshopID, orderID, position, photoData, caption)
}

func validStagePhoto(photoData string) bool {
	const dataURLPrefix = "data:image/jpeg;base64,"
	encoded := strings.TrimPrefix(strings.TrimSpace(photoData), dataURLPrefix)
	if encoded == strings.TrimSpace(photoData) || encoded == "" || len(encoded) > base64.StdEncoding.EncodedLen(maxStagePhotoBytes) {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(decoded) == 0 || len(decoded) > maxStagePhotoBytes {
		return false
	}
	config, err := jpeg.DecodeConfig(bytes.NewReader(decoded))
	return err == nil && config.Width > 0 && config.Height > 0 && config.Width <= 4096 && config.Height <= 4096
}

func (s *Service) GetDashboardMetrics(ctx context.Context, workshopID uuid.UUID) (*DashboardMetrics, error) {
	return s.repo.GetDashboardMetrics(ctx, workshopID)
}
