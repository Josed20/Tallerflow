package orders

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/jpeg"
	"strings"
	"time"

	"github.com/google/uuid"
)

const maxStagePhotoBytes = 3 * 1024 * 1024

type Service struct {
	repo   Repository
	photos PhotoStore
}

type PhotoStore interface {
	Put(ctx context.Context, key string, photo []byte) error
	URL(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}

func NewService(repo Repository, photos ...PhotoStore) *Service {
	service := &Service{repo: repo}
	if len(photos) > 0 {
		service.photos = photos[0]
	}
	return service
}

func (s *Service) List(ctx context.Context, workshopID uuid.UUID) ([]Order, error) {
	return s.repo.List(ctx, workshopID)
}

func (s *Service) GetByID(ctx context.Context, workshopID uuid.UUID, id uuid.UUID) (*Order, error) {
	order, err := s.repo.GetByID(ctx, workshopID, id)
	if err != nil {
		return nil, err
	}
	for i := range order.Photos {
		if err := resolvePhoto(ctx, s.photos, &order.Photos[i].PhotoData, order.Photos[i].ObjectKey); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func (s *Service) Create(ctx context.Context, workshopID uuid.UUID, input CreateOrderInput, idempotencyKey string) (*Order, error) {
	return s.repo.Create(ctx, workshopID, input, idempotencyKey)
}

func (s *Service) Update(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, actorID uuid.UUID, input UpdateOrderInput) (*Order, error) {
	return s.repo.Update(ctx, workshopID, orderID, actorID, input)
}

func (s *Service) UpdateStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, status string) (*OrderStage, error) {
	return s.repo.UpdateStage(ctx, workshopID, orderID, position, actorID, status)
}

func (s *Service) UndoStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID) (*OrderStage, error) {
	return s.repo.UndoStage(ctx, workshopID, orderID, position, actorID)
}

func (s *Service) Cancel(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, actorID uuid.UUID, reason string) (*Order, error) {
	return s.repo.Cancel(ctx, workshopID, orderID, actorID, reason)
}

func (s *Service) ReopenStage(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, actorID uuid.UUID, reason string) (*OrderStage, error) {
	return s.repo.ReopenStage(ctx, workshopID, orderID, position, actorID, reason)
}

func (s *Service) AddStagePhoto(ctx context.Context, workshopID uuid.UUID, orderID uuid.UUID, position int, photoData string, caption string) (*StagePhoto, error) {
	if !validStagePhoto(photoData) || len([]rune(strings.TrimSpace(caption))) > 300 {
		return nil, ErrOrderInvalid
	}
	if s.photos == nil {
		return s.repo.AddStagePhoto(ctx, workshopID, orderID, position, photoData, "", caption)
	}
	encoded := strings.TrimPrefix(strings.TrimSpace(photoData), "data:image/jpeg;base64,")
	decoded, _ := base64.StdEncoding.DecodeString(encoded)
	key := fmt.Sprintf("workshops/%s/orders/%s/photos/%s.jpg", workshopID, orderID, uuid.New())
	if err := s.photos.Put(ctx, key, decoded); err != nil {
		return nil, err
	}
	photo, err := s.repo.AddStagePhoto(ctx, workshopID, orderID, position, "", key, caption)
	if err != nil {
		// The SQL insert can fail after the upload; remove the unreferenced object.
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = s.photos.Delete(cleanupCtx, key)
		return nil, err
	}
	if err := resolvePhoto(ctx, s.photos, &photo.PhotoData, photo.ObjectKey); err != nil {
		return nil, err
	}
	return photo, nil
}

func resolvePhoto(ctx context.Context, store PhotoStore, data *string, key string) error {
	if key == "" {
		return nil
	}
	if store == nil {
		return errors.New("photo store is unavailable")
	}
	url, err := store.URL(ctx, key)
	if err == nil {
		*data = url
	}
	return err
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
