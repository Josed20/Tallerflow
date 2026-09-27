package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrBootstrapAlreadyExists = errors.New("BOOTSTRAP_ALREADY_EXISTS")
	ErrBootstrapInvalidInput  = errors.New("BOOTSTRAP_INVALID_INPUT")
)

type BootstrapInput struct {
	Email        string
	Name         string
	WorkshopName string
	Password     string
}

type BootstrapResult struct {
	UserID             uuid.UUID
	WorkshopID         uuid.UUID
	Role               string
	MustChangePassword bool
}

type WebBootstrapResult struct {
	BootstrapResult
	Token     string
	CSRFToken string
	ExpiresAt time.Time
}

// BootstrapTx is the privileged, atomic write boundary for the first OWNER.
// Implementations must roll back all writes (including audit) on any error.
type BootstrapTx interface {
	InsertUser(context.Context, string, string) (uuid.UUID, error)
	ClaimInitialOwner(context.Context, uuid.UUID) error
	InsertCredential(context.Context, uuid.UUID, string, bool, *time.Time) error
	InsertWorkshop(context.Context, string, string) (uuid.UUID, error)
	InsertMembership(context.Context, uuid.UUID, uuid.UUID, string) error
	InsertAudit(context.Context, string, uuid.UUID, uuid.UUID) error
}

type BootstrapStore interface {
	WithinTransaction(context.Context, func(BootstrapTx) error) error
}

type BootstrapService struct {
	store  BootstrapStore
	hasher PasswordHasher
	clock  func() time.Time
}

func NewBootstrapService(store BootstrapStore, hasher PasswordHasher) *BootstrapService {
	return NewBootstrapServiceWithClock(store, hasher, time.Now)
}

func NewBootstrapServiceWithClock(store BootstrapStore, hasher PasswordHasher, clock func() time.Time) *BootstrapService {
	if clock == nil {
		clock = time.Now
	}
	return &BootstrapService{store: store, hasher: hasher, clock: clock}
}

func (s *BootstrapService) CreateOwner(ctx context.Context, in BootstrapInput) (BootstrapResult, error) {
	return s.createOwner(ctx, in, true)
}

func (s *BootstrapService) CreateWebOwner(ctx context.Context, in BootstrapInput, metadata SessionMetadata, sessions *SessionService) (WebBootstrapResult, error) {
	if sessions == nil {
		return WebBootstrapResult{}, errors.New("bootstrap service is not configured")
	}
	result, err := s.createOwner(ctx, in, false)
	if err != nil {
		return WebBootstrapResult{}, err
	}
	created, err := sessions.Create(ctx, result.UserID, metadata)
	if err != nil {
		return WebBootstrapResult{}, fmt.Errorf("create onboarding session: %w", err)
	}
	return WebBootstrapResult{
		BootstrapResult: result,
		Token:           created.Token, CSRFToken: created.CSRFToken, ExpiresAt: created.ExpiresAt,
	}, nil
}

func (s *BootstrapService) createOwner(ctx context.Context, in BootstrapInput, mustChangePassword bool) (BootstrapResult, error) {
	var result BootstrapResult
	if s == nil || s.store == nil || s.clock == nil {
		return result, errors.New("bootstrap service is not configured")
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	in.WorkshopName = strings.TrimSpace(in.WorkshopName)
	if in.Email == "" || !strings.Contains(in.Email, "@") || in.Name == "" || in.WorkshopName == "" || len(in.Password) < 12 || len(in.Password) > 1<<20 {
		return result, ErrBootstrapInvalidInput
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return result, fmt.Errorf("hash initial credential: %w", err)
	}
	err = s.store.WithinTransaction(ctx, func(tx BootstrapTx) error {
		userID, err := tx.InsertUser(ctx, in.Email, in.Name)
		if err != nil {
			return err
		}
		if err := tx.ClaimInitialOwner(ctx, userID); err != nil {
			return err
		}
		var passwordChangedAt *time.Time
		if !mustChangePassword {
			changedAt := s.clock().UTC()
			passwordChangedAt = &changedAt
		}
		if err := tx.InsertCredential(ctx, userID, hash, mustChangePassword, passwordChangedAt); err != nil {
			return err
		}
		workshopID, err := tx.InsertWorkshop(ctx, in.WorkshopName, "America/Lima")
		if err != nil {
			return err
		}
		if err := tx.InsertMembership(ctx, userID, workshopID, "OWNER"); err != nil {
			return err
		}
		if err := tx.InsertAudit(ctx, "OWNER_BOOTSTRAPPED", userID, workshopID); err != nil {
			return err
		}
		result = BootstrapResult{
			UserID: userID, WorkshopID: workshopID, Role: "OWNER", MustChangePassword: mustChangePassword,
		}
		return nil
	})
	if err != nil {
		return BootstrapResult{}, err
	}
	return result, nil
}
