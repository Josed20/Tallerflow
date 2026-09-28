package passwordreset

import (
	"context"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type ServiceConfig struct {
	BaseURL       string
	TokenLifetime time.Duration
}

type Service struct {
	repo     Repository
	delivery PasswordResetDelivery
	hasher   PasswordHasher
	config   ServiceConfig
	clock    func() time.Time
}

func NewService(repo Repository, delivery PasswordResetDelivery, hasher PasswordHasher, cfg ServiceConfig, clock func() time.Time) *Service {
	if cfg.TokenLifetime <= 0 {
		cfg.TokenLifetime = 1 * time.Hour
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://localhost:8080"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if clock == nil {
		clock = time.Now
	}
	return &Service{
		repo:     repo,
		delivery: delivery,
		hasher:   hasher,
		config:   cfg,
		clock:    clock,
	}
}

func (s *Service) RequestReset(ctx context.Context, email, ip string) error {
	trimmedEmail := strings.ToLower(strings.TrimSpace(email))
	if trimmedEmail == "" || !isValidEmail(trimmedEmail) {
		return fmt.Errorf("invalid email format")
	}

	allowed, err := s.repo.AllowRequest(ctx, ip, trimmedEmail)
	if err != nil {
		return fmt.Errorf("check rate limit: %w", err)
	}
	if !allowed {
		return ErrRateLimited
	}

	userID, exists, err := s.repo.FindUserByEmail(ctx, trimmedEmail)
	if err != nil {
		return fmt.Errorf("lookup user by email: %w", err)
	}

	// Anti-enumeration: if user does not exist, return nil without disclosing account existence
	if !exists {
		return nil
	}

	rawToken, tokenHash, err := GenerateToken()
	if err != nil {
		return fmt.Errorf("generate reset token: %w", err)
	}

	now := s.clock().UTC()
	expiresAt := now.Add(s.config.TokenLifetime)

	if err := s.repo.CreateResetToken(ctx, userID, tokenHash, expiresAt, now); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.BaseURL, rawToken)
	if err := s.delivery.Deliver(ctx, trimmedEmail, resetURL); err != nil {
		return fmt.Errorf("deliver reset email: %w", err)
	}

	return nil
}

func (s *Service) ConsumeReset(ctx context.Context, rawToken, newPassword string) error {
	trimmedToken := strings.TrimSpace(rawToken)
	if trimmedToken == "" {
		return ErrTokenInvalid
	}

	if len(newPassword) < 12 {
		return ErrPasswordTooWeak
	}

	tokenHash, err := HashToken(trimmedToken)
	if err != nil {
		return ErrTokenInvalid
	}

	newPasswordHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}

	now := s.clock().UTC()
	if err := s.repo.ConsumeResetTokenAndChangePassword(ctx, tokenHash, newPasswordHash, now); err != nil {
		return err
	}

	return nil
}

func isValidEmail(value string) bool {
	if len(value) == 0 || len(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
