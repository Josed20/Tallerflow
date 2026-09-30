package onboarding

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/google/uuid"
)

var ErrUnavailable = errors.New("onboarding is unavailable")

type Status struct {
	Available bool
}

type Input struct {
	WorkshopName         string
	OwnerName            string
	Email                string
	Password             string
	PasswordConfirmation string
}

type Result struct {
	UserID             uuid.UUID
	WorkshopID         uuid.UUID
	Role               string
	MustChangePassword bool
	Token              string
	CSRFToken          string
	ExpiresAt          time.Time
}

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "onboarding input is invalid" }

type StatusStore interface {
	Claimed(context.Context) (bool, error)
}

type OwnerCreator interface {
	CreateWebOwner(context.Context, auth.BootstrapInput, auth.SessionMetadata) (auth.WebBootstrapResult, error)
}

type OwnerCreatorFunc func(context.Context, auth.BootstrapInput, auth.SessionMetadata) (auth.WebBootstrapResult, error)

func (f OwnerCreatorFunc) CreateWebOwner(ctx context.Context, input auth.BootstrapInput, metadata auth.SessionMetadata) (auth.WebBootstrapResult, error) {
	return f(ctx, input, metadata)
}

type Service struct {
	status  StatusStore
	creator OwnerCreator
}

func NewService(status StatusStore, creator OwnerCreator) *Service {
	return &Service{status: status, creator: creator}
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	if s == nil || s.status == nil {
		return Status{}, errors.New("onboarding status store is not configured")
	}
	claimed, err := s.status.Claimed(ctx)
	if err != nil {
		return Status{}, err
	}
	return Status{Available: !claimed}, nil
}

func (s *Service) Create(ctx context.Context, input Input, metadata auth.SessionMetadata) (Result, error) {
	if s == nil || s.creator == nil {
		return Result{}, errors.New("onboarding owner creator is not configured")
	}
	input.WorkshopName = strings.TrimSpace(input.WorkshopName)
	input.OwnerName = strings.TrimSpace(input.OwnerName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if fields := validate(input); len(fields) > 0 {
		return Result{}, &ValidationError{Fields: fields}
	}
	created, err := s.creator.CreateWebOwner(ctx, auth.BootstrapInput{
		Email: input.Email, Name: input.OwnerName, WorkshopName: input.WorkshopName, Password: input.Password,
	}, metadata)
	if errors.Is(err, auth.ErrBootstrapAlreadyExists) {
		return Result{}, ErrUnavailable
	}
	if err != nil {
		return Result{}, err
	}
	return Result{
		UserID: created.UserID, WorkshopID: created.WorkshopID, Role: created.Role,
		MustChangePassword: created.MustChangePassword, Token: created.Token,
		CSRFToken: created.CSRFToken, ExpiresAt: created.ExpiresAt,
	}, nil
}

func validate(input Input) map[string]string {
	fields := make(map[string]string)
	if len(input.WorkshopName) == 0 || len(input.WorkshopName) > 160 {
		fields["workshop_name"] = "Workshop name is required and must contain at most 160 characters."
	}
	if len(input.OwnerName) == 0 || len(input.OwnerName) > 160 {
		fields["owner_name"] = "Owner name is required and must contain at most 160 characters."
	}
	if !validEmail(input.Email) {
		fields["email"] = "Email is invalid."
	}
	if len(input.Password) < 12 || len(input.Password) > 1<<20 {
		fields["password"] = "Password must contain between 12 and 1048576 characters."
	}
	if input.PasswordConfirmation != input.Password {
		fields["password_confirmation"] = "Password confirmation does not match."
	}
	return fields
}

func validEmail(value string) bool {
	if len(value) == 0 || len(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
