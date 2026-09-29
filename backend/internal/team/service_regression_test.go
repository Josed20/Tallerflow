package team

import (
	"context"
	"testing"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
)

func TestConsumeRejectsInvalidProfileBeforeTokenLookup(t *testing.T) {
	service, err := NewService(&consumeStoreSpy{}, auth.NewPasswordHasher(auth.DefaultPasswordParams()), []byte("pepper"), "http://localhost:8080", nil, nil)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	_, err = service.Consume(context.Background(), ConsumeInput{
		Token:    "VHVKYWpXTllUSWVSc09KU2hwU3dPUUdPbG5IY3JvRGFGb0h6SWtfQ0hXdw",
		Name:     "Demo User",
		Password: "short",
	})

	if err != ErrInvalidInput {
		t.Fatalf("Consume() error = %v, want ErrInvalidInput", err)
	}
}

type consumeStoreSpy struct {
	Store
}
