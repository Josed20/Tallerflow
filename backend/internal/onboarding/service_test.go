package onboarding

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/google/uuid"
)

func TestServiceStatusReturnsOnlyPublicAvailability(t *testing.T) {
	service := NewService(statusStoreFake{claimed: false}, OwnerCreatorFunc(func(context.Context, auth.BootstrapInput, auth.SessionMetadata) (auth.WebBootstrapResult, error) {
		return auth.WebBootstrapResult{}, nil
	}))

	status, err := service.Status(context.Background())

	if err != nil || !status.Available {
		t.Fatalf("Status() = %+v, %v, want available", status, err)
	}
}

func TestServiceCreateNormalizesInputAndReturnsFinalSession(t *testing.T) {
	wantUserID, wantWorkshopID := uuid.New(), uuid.New()
	creator := &ownerCreatorFake{result: auth.WebBootstrapResult{
		BootstrapResult: auth.BootstrapResult{UserID: wantUserID, WorkshopID: wantWorkshopID, Role: "OWNER", MustChangePassword: false},
		Token:           "opaque", CSRFToken: "csrf", ExpiresAt: time.Date(2026, 9, 27, 23, 0, 0, 0, time.UTC),
	}}
	service := NewService(statusStoreFake{}, creator)

	result, err := service.Create(context.Background(), Input{
		WorkshopName: " Taller Demo ", OwnerName: " José Dueño ", Email: " OWNER@Example.com ",
		Password: "Secure password 123!", PasswordConfirmation: "Secure password 123!",
	}, auth.SessionMetadata{IPPrefix: "203.0.113.10", UserAgent: "browser"})

	if err != nil {
		t.Fatal(err)
	}
	if creator.input.Email != "owner@example.com" || creator.input.Name != "José Dueño" || creator.input.WorkshopName != "Taller Demo" {
		t.Fatalf("Create() did not normalize input: %+v", creator.input)
	}
	if result.UserID != wantUserID || result.WorkshopID != wantWorkshopID || result.Token != "opaque" || result.MustChangePassword {
		t.Fatalf("Create() returned unexpected result: %+v", result)
	}
}

func TestServiceCreateRejectsInvalidFieldsBeforeHashingOrWriting(t *testing.T) {
	creator := &ownerCreatorFake{}
	service := NewService(statusStoreFake{}, creator)

	_, err := service.Create(context.Background(), Input{
		WorkshopName: "", OwnerName: "", Email: "not-an-email", Password: "short", PasswordConfirmation: "different",
	}, auth.SessionMetadata{})

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Create() error = %v, want ValidationError", err)
	}
	for _, field := range []string{"workshop_name", "owner_name", "email", "password", "password_confirmation"} {
		if validation.Fields[field] == "" {
			t.Fatalf("validation did not report %s: %+v", field, validation.Fields)
		}
	}
	if creator.calls != 0 {
		t.Fatal("invalid input reached the privileged owner creator")
	}
}

func TestServiceCreatePreservesClaimedInstallationConflict(t *testing.T) {
	creator := &ownerCreatorFake{err: auth.ErrBootstrapAlreadyExists}
	service := NewService(statusStoreFake{}, creator)

	_, err := service.Create(context.Background(), validInput(), auth.SessionMetadata{})

	if !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create() error = %v, want ErrUnavailable", err)
	}
}

func validInput() Input {
	return Input{WorkshopName: "Taller Demo", OwnerName: "Owner Demo", Email: "owner@example.com", Password: "Secure password 123!", PasswordConfirmation: "Secure password 123!"}
}

type statusStoreFake struct {
	claimed bool
	err     error
}

func (f statusStoreFake) Claimed(context.Context) (bool, error) { return f.claimed, f.err }

type ownerCreatorFake struct {
	input  auth.BootstrapInput
	result auth.WebBootstrapResult
	err    error
	calls  int
}

func (f *ownerCreatorFake) CreateWebOwner(_ context.Context, input auth.BootstrapInput, _ auth.SessionMetadata) (auth.WebBootstrapResult, error) {
	f.calls++
	f.input = input
	return f.result, f.err
}
