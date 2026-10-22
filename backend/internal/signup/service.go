package signup

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	platformmail "github.com/Josed20/Tallerflow/backend/platform/mail"
	"github.com/google/uuid"
)

type PasswordHasher interface {
	Hash(string) (string, error)
}

type SessionCreator interface {
	Create(context.Context, uuid.UUID, auth.SessionMetadata) (auth.RawSession, error)
}

type Config struct {
	Enabled bool
	// BaseURL is the public web origin the verification link points to.
	BaseURL       string
	TokenLifetime time.Duration
	// MinimumDuration pads register and resend so a new address and a known
	// one take the same time to answer.
	MinimumDuration time.Duration
	DeliveryTimeout time.Duration
}

type Service struct {
	repo     Repository
	hasher   PasswordHasher
	sessions SessionCreator
	mailer   platformmail.Sender
	config   Config
	clock    func() time.Time
	// dispatch runs email delivery; tests replace it to send synchronously.
	dispatch func(func())
}

func NewService(repo Repository, hasher PasswordHasher, sessions SessionCreator, mailer platformmail.Sender, cfg Config, clock func() time.Time) (*Service, error) {
	if repo == nil || hasher == nil || sessions == nil || mailer == nil {
		return nil, errors.New("signup repository, hasher, sessions and mailer are required")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("signup base URL is required")
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if cfg.TokenLifetime <= 0 {
		cfg.TokenLifetime = 24 * time.Hour
	}
	if cfg.MinimumDuration <= 0 {
		cfg.MinimumDuration = 400 * time.Millisecond
	}
	if cfg.DeliveryTimeout <= 0 {
		cfg.DeliveryTimeout = 15 * time.Second
	}
	if clock == nil {
		clock = time.Now
	}
	return &Service{
		repo: repo, hasher: hasher, sessions: sessions, mailer: mailer, config: cfg, clock: clock,
		dispatch: func(send func()) { go send() },
	}, nil
}

// Register accepts a new workshop. Whether or not the email already has an
// account, a valid request returns nil after the same minimum time.
func (s *Service) Register(ctx context.Context, input Input) error {
	if !s.config.Enabled {
		return ErrClosed
	}
	input.WorkshopName = strings.TrimSpace(input.WorkshopName)
	input.OwnerName = strings.TrimSpace(input.OwnerName)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if fields := validate(input); len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}

	startedAt := time.Now()
	defer s.waitForMinimumDuration(ctx, startedAt)

	// The password is hashed on every path, so a known address cannot be
	// told apart by skipping the expensive step.
	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return fmt.Errorf("hash signup password: %w", err)
	}
	rawToken, tokenHash, err := GenerateToken()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}
	issued, err := s.repo.Register(ctx, PendingAccount{
		Email: input.Email, OwnerName: input.OwnerName, WorkshopName: input.WorkshopName,
		PasswordHash: passwordHash, TokenHash: tokenHash, ExpiresAt: s.clock().UTC().Add(s.config.TokenLifetime),
	})
	if err != nil {
		return fmt.Errorf("register workshop: %w", err)
	}
	if issued {
		s.sendVerification(input.Email, Recipient{Name: input.OwnerName, WorkshopName: input.WorkshopName}, rawToken)
	}
	return nil
}

// Resend issues a new link for an unverified account and answers the same
// for unknown, verified and pending addresses.
func (s *Service) Resend(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if !validEmail(email) {
		return &ValidationError{Fields: []string{"email"}}
	}

	startedAt := time.Now()
	defer s.waitForMinimumDuration(ctx, startedAt)

	rawToken, tokenHash, err := GenerateToken()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}
	recipient, issued, err := s.repo.ReissueToken(ctx, email, tokenHash, s.clock().UTC().Add(s.config.TokenLifetime))
	if err != nil {
		return fmt.Errorf("reissue verification token: %w", err)
	}
	if issued {
		s.sendVerification(email, recipient, rawToken)
	}
	return nil
}

// Verify consumes the emailed token and opens a browser session for the owner.
func (s *Service) Verify(ctx context.Context, rawToken string, metadata auth.SessionMetadata) (auth.RawSession, error) {
	tokenHash, ok := HashToken(strings.TrimSpace(rawToken))
	if !ok {
		return auth.RawSession{}, ErrTokenInvalid
	}
	userID, err := s.repo.ConsumeToken(ctx, tokenHash)
	if err != nil {
		return auth.RawSession{}, err
	}
	metadata.Kind = auth.SessionKindWeb
	session, err := s.sessions.Create(ctx, userID, metadata)
	if err != nil {
		return auth.RawSession{}, fmt.Errorf("create verified session: %w", err)
	}
	return session, nil
}

func (s *Service) sendVerification(email string, recipient Recipient, rawToken string) {
	link := s.config.BaseURL + "/verificar-correo?token=" + rawToken
	s.dispatch(func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.DeliveryTimeout)
		defer cancel()
		msg, err := platformmail.Render("verify-email", map[string]string{
			"Name": recipient.Name, "WorkshopName": recipient.WorkshopName, "URL": link, "ExpiresIn": "24 horas",
		})
		if err != nil {
			log.Printf("verification email render failed")
			return
		}
		msg.To = email
		// The owner can ask for a new link, so a failed send is only logged,
		// without the address or the token.
		if err := s.mailer.Send(ctx, msg); err != nil {
			log.Printf("verification email delivery failed")
		}
	})
}

func (s *Service) waitForMinimumDuration(ctx context.Context, startedAt time.Time) {
	remaining := s.config.MinimumDuration - time.Since(startedAt)
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

func validate(input Input) []string {
	fields := make([]string, 0, 5)
	if len(input.WorkshopName) == 0 || len([]rune(input.WorkshopName)) > 160 {
		fields = append(fields, "workshop_name")
	}
	if len(input.OwnerName) == 0 || len([]rune(input.OwnerName)) > 160 {
		fields = append(fields, "owner_name")
	}
	if !validEmail(input.Email) {
		fields = append(fields, "email")
	}
	if len(input.Password) < 12 || len(input.Password) > 1<<20 {
		fields = append(fields, "password")
	}
	if !input.AcceptTerms {
		fields = append(fields, "accept_terms")
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
