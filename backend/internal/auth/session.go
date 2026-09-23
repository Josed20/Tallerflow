package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"io"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/google/uuid"
)

const (
	sessionTokenBytes = 32
	sessionLifetime   = 8 * time.Hour
	maxRawTokenBytes  = 1024
)

type SessionService struct {
	repository SessionRepository
	pepper     []byte
	clock      func() time.Time
	random     io.Reader
}

func NewSessionService(repository SessionRepository, pepper []byte, clock func() time.Time, randomSource io.Reader) *SessionService {
	if clock == nil {
		clock = time.Now
	}
	if randomSource == nil {
		randomSource = rand.Reader
	}
	return &SessionService{
		repository: repository,
		pepper:     append([]byte(nil), pepper...),
		clock:      clock,
		random:     randomSource,
	}
}

func (s *SessionService) Create(ctx context.Context, userID uuid.UUID, metadata SessionMetadata) (RawSession, error) {
	if err := s.validateConfiguration(); err != nil {
		return RawSession{}, err
	}

	token, err := security.RandomToken(s.random, sessionTokenBytes)
	if err != nil {
		return RawSession{}, err
	}
	now := s.clock().UTC()
	csrfTokenHash := sha256.Sum256([]byte(security.DeriveCSRFToken(s.pepper, token)))
	created := NewSession{
		UserID:        userID,
		TokenHash:     security.TokenDigest(s.pepper, token),
		CSRFTokenHash: csrfTokenHash,
		Metadata:      metadata,
		CreatedAt:     now,
		ExpiresAt:     now.Add(sessionLifetime),
	}
	session, err := s.repository.Insert(ctx, created)
	if err != nil {
		return RawSession{}, err
	}
	return RawSession{Token: token, ExpiresAt: created.ExpiresAt, Session: session}, nil
}

func (s *SessionService) Authenticate(ctx context.Context, rawToken string) (Session, error) {
	if err := s.validateConfiguration(); err != nil {
		return Session{}, err
	}
	if len(rawToken) == 0 || len(rawToken) > maxRawTokenBytes {
		return Session{}, ErrSessionInvalid
	}

	now := s.clock().UTC()
	session, err := s.repository.FindActiveByTokenHash(ctx, security.TokenDigest(s.pepper, rawToken), now)
	if err != nil {
		return Session{}, err
	}
	if session.RevokedAt != nil || !session.ExpiresAt.After(now) {
		return Session{}, ErrSessionInvalid
	}
	return session, nil
}

func (s *SessionService) Revoke(ctx context.Context, sessionID uuid.UUID) error {
	if err := s.validateConfiguration(); err != nil {
		return err
	}
	return s.repository.Revoke(ctx, sessionID, s.clock().UTC())
}

func (s *SessionService) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if err := s.validateConfiguration(); err != nil {
		return err
	}
	return s.repository.RevokeAllForUser(ctx, userID, s.clock().UTC())
}

func (s *SessionService) validateConfiguration() error {
	if s == nil || s.repository == nil || len(s.pepper) == 0 || s.clock == nil || s.random == nil {
		return ErrSessionConfiguration
	}
	return nil
}
