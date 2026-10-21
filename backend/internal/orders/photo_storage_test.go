package orders

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"image"
	"image/jpeg"
	"testing"

	"github.com/google/uuid"
)

type photoRepository struct {
	Repository
	objectKey string
	photoData string
	insertErr error
	order     *Order
}

func (r *photoRepository) AddStagePhoto(_ context.Context, workshopID, orderID uuid.UUID, position int, data, key, caption string) (*StagePhoto, error) {
	r.photoData, r.objectKey = data, key
	if r.insertErr != nil {
		return nil, r.insertErr
	}
	return &StagePhoto{WorkshopID: workshopID, OrderID: orderID, StagePosition: position, PhotoData: data, ObjectKey: key, Caption: caption}, nil
}

func (r *photoRepository) GetByID(context.Context, uuid.UUID, uuid.UUID) (*Order, error) {
	return r.order, nil
}

type photoStore struct {
	putKey, deletedKey string
	putData            []byte
}

func (s *photoStore) Put(_ context.Context, key string, data []byte) error {
	s.putKey, s.putData = key, data
	return nil
}
func (s *photoStore) URL(_ context.Context, key string) (string, error) {
	return "https://account.r2.cloudflarestorage.com/bucket/" + key, nil
}
func (s *photoStore) Delete(_ context.Context, key string) error { s.deletedKey = key; return nil }

func jpegDataURL(t *testing.T) string {
	t.Helper()
	var imageBytes bytes.Buffer
	if err := jpeg.Encode(&imageBytes, image.NewRGBA(image.Rect(0, 0, 2, 2)), nil); err != nil {
		t.Fatal(err)
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageBytes.Bytes())
}

func TestNewPhotoStoresOnlyObjectKeyAndReturnsTemporaryURL(t *testing.T) {
	repo := &photoRepository{}
	store := &photoStore{}
	photo, err := NewService(repo, store).AddStagePhoto(context.Background(), uuid.New(), uuid.New(), 1, jpegDataURL(t), "Costura")
	if err != nil {
		t.Fatal(err)
	}
	if repo.photoData != "" || repo.objectKey == "" || len(store.putData) == 0 || store.putKey != repo.objectKey {
		t.Fatalf("stored data=%q, key=%q, uploaded bytes=%d", repo.photoData, repo.objectKey, len(store.putData))
	}
	if photo.PhotoData == "" || photo.ObjectKey != repo.objectKey {
		t.Fatal("photo response lacks the temporary URL or object key")
	}
}

func TestNewPhotoRemovesObjectWhenDatabaseInsertFails(t *testing.T) {
	repo := &photoRepository{insertErr: errors.New("insert failed")}
	store := &photoStore{}
	_, err := NewService(repo, store).AddStagePhoto(context.Background(), uuid.New(), uuid.New(), 1, jpegDataURL(t), "")
	if err == nil || store.putKey == "" || store.deletedKey != store.putKey {
		t.Fatalf("error=%v uploaded=%q deleted=%q", err, store.putKey, store.deletedKey)
	}
}

func TestGetOrderKeepsLegacyPhotoAndSignsNewPhoto(t *testing.T) {
	repo := &photoRepository{order: &Order{Photos: []StagePhoto{
		{PhotoData: "data:image/jpeg;base64,old"}, {ObjectKey: "workshops/new.jpg"},
	}}}
	order, err := NewService(repo, &photoStore{}).GetByID(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if order.Photos[0].PhotoData != "data:image/jpeg;base64,old" || order.Photos[1].PhotoData == "" {
		t.Fatalf("unexpected photo representations: %+v", order.Photos)
	}
}
