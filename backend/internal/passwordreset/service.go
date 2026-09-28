package passwordreset

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

type PasswordHasher interface {
	Hash(password string) (string, error)
}

type ServiceConfig struct {
	BaseURL                string
	TokenLifetime          time.Duration
	DeliveryTimeout        time.Duration
	RequestMinimumDuration time.Duration
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
	if cfg.DeliveryTimeout <= 0 {
		cfg.DeliveryTimeout = 10 * time.Second
	}
	if cfg.RequestMinimumDuration <= 0 {
		cfg.RequestMinimumDuration = 100 * time.Millisecond
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
	startedAt := time.Now()
	defer s.waitForMinimumRequestDuration(ctx, startedAt)

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
		return fmt.Errorf("find recovery user: %w", err)
	}

	// Anti-enumeration: if user does not exist, return nil without disclosing account existence
	if !exists {
		return nil
	}

	rawToken, tokenHash, err := GenerateToken()
	if err != nil {
		return fmt.Errorf("generate recovery token: %w", err)
	}

	now := s.clock().UTC()
	expiresAt := now.Add(s.config.TokenLifetime)

	if err := s.repo.CreateResetToken(ctx, userID, tokenHash, expiresAt, now); err != nil {
		return fmt.Errorf("create recovery token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.config.BaseURL, rawToken)
	s.dispatchDelivery(trimmedEmail, resetURL)

	return nil
}

func (s *Service) ConsumeReset(ctx context.Context, rawToken, newPassword string, clientIP ...string) error {
	trimmedToken := strings.TrimSpace(rawToken)
	if trimmedToken == "" {
		return ErrTokenInvalid
	}

	if len(newPassword) < 12 {
		return ErrPasswordTooWeak
	}

	requestIP := ""
	if len(clientIP) > 0 {
		requestIP = strings.TrimSpace(clientIP[0])
	}
	rateTokenHash := sha256.Sum256([]byte(trimmedToken))
	allowed, err := s.repo.AllowConsume(ctx, requestIP, rateTokenHash[:])
	if err != nil {
		return fmt.Errorf("check consume rate limit: %w", err)
	}
	if !allowed {
		return ErrRateLimited
	}

	tokenHash, err := HashToken(trimmedToken)
	if err != nil {
		return ErrTokenInvalid
	}

	now := s.clock().UTC()
	if err := s.repo.ConsumeResetTokenAndChangePassword(ctx, tokenHash, now, func() (string, error) {
		return s.hasher.Hash(newPassword)
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) dispatchDelivery(recipientEmail, resetURL string) {
	if s.delivery == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.DeliveryTimeout)
		defer cancel()
		_ = s.delivery.Deliver(ctx, recipientEmail, resetURL)
	}()
}

func (s *Service) waitForMinimumRequestDuration(ctx context.Context, startedAt time.Time) {
	remaining := s.config.RequestMinimumDuration - time.Since(startedAt)
	if remaining <= 0 {
		return
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
	}
}

func isValidEmail(value string) bool {
	if len(value) == 0 || len(value) > 254 {
		return false
	}
	address, err := mail.ParseAddress(value)
	return err == nil && address.Address == value
}
