package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/google/uuid"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTooManyAttempts    = errors.New("too many login attempts")
	ErrPasswordInvalid    = errors.New("current password is invalid")
	ErrPasswordPolicy     = errors.New("new password violates policy")
	ErrPasswordReuse      = errors.New("new password must differ from current password")
	ErrCredentialNotFound = errors.New("credential not found")
)

var nonexistentCredentialHash struct {
	sync.Once
	value string
	err   error
}

type Credential struct {
	UserID             uuid.UUID
	PasswordHash       string
	Active             bool
	MustChangePassword bool
}

type CredentialRepository interface {
	FindByEmail(context.Context, string) (*Credential, error)
}

// PasswordCredentialRepository is needed by restoration and password changes.
// It is separate so login-only adapters can stay small.
type PasswordCredentialRepository interface {
	CredentialRepository
	FindByUserID(context.Context, uuid.UUID) (*Credential, error)
	// The adapter updates the hash, clears must-change, and revokes all user
	// sessions in one database transaction. An error leaves all three intact.
	ChangePasswordAndRevokeSessions(context.Context, uuid.UUID, string, time.Time) error
}

type MembershipResolver interface {
	ResolveActive(context.Context, uuid.UUID) (ActiveMembership, error)
}

type LoginLimiter interface {
	Allow(context.Context, string, string) (bool, error)
	RecordFailure(context.Context, string, string) error
}

type AuthSession struct {
	Token              string
	CSRFToken          string
	ExpiresAt          time.Time
	MustChangePassword bool
}

type AuthService struct {
	credentials CredentialRepository
	memberships MembershipResolver
	sessions    *SessionService
	hasher      PasswordHasher
	limiter     LoginLimiter
	csrfSecret  []byte
}

// A purpose label in DeriveCSRFToken separates CSRF values from session-token
// digests even though both use the server's session pepper.
func NewAuthService(credentials CredentialRepository, memberships MembershipResolver, sessions *SessionService, hasher PasswordHasher, limiter LoginLimiter) *AuthService {
	service := &AuthService{credentials: credentials, memberships: memberships, sessions: sessions, hasher: hasher, limiter: limiter}
	if sessions != nil {
		service.csrfSecret = append([]byte(nil), sessions.pepper...)
	}
	return service
}

func (s *AuthService) Login(ctx context.Context, email, password, clientIP string) (AuthSession, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" || len(password) > 1<<20 {
		return AuthSession{}, ErrInvalidCredentials
	}
	if s.limiter != nil {
		allowed, err := s.limiter.Allow(ctx, email, clientIP)
		if err != nil {
			return AuthSession{}, err
		}
		if !allowed {
			return AuthSession{}, ErrTooManyAttempts
		}
	}
	credential, err := s.credentials.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, ErrCredentialNotFound) {
		return AuthSession{}, err
	}
	if errors.Is(err, ErrCredentialNotFound) {
		credential = nil
	}
	valid := false
	verificationHash := ""
	if credential != nil {
		verificationHash = credential.PasswordHash
	} else {
		nonexistentCredentialHash.Do(func() {
			nonexistentCredentialHash.value, nonexistentCredentialHash.err = s.hasher.Hash("tallerflow nonexistent account sentinel")
		})
		if nonexistentCredentialHash.err != nil {
			return AuthSession{}, nonexistentCredentialHash.err
		}
		verificationHash = nonexistentCredentialHash.value
	}
	match, verifyErr := s.hasher.Verify(verificationHash, password)
	if verifyErr == nil && credential != nil && credential.Active {
		valid = match
	}
	if !valid {
		return AuthSession{}, s.recordInvalidLogin(ctx, email, clientIP)
	}
	_, err = s.memberships.ResolveActive(ctx, credential.UserID)
	if err != nil {
		if !errors.Is(err, ErrInvalidCredentials) {
			return AuthSession{}, err
		}
		return AuthSession{}, s.recordInvalidLogin(ctx, email, clientIP)
	}
	if len(s.csrfSecret) == 0 {
		return AuthSession{}, errors.New("CSRF secret is not configured")
	}
	if err := s.sessions.RevokeAllForUser(ctx, credential.UserID); err != nil {
		return AuthSession{}, err
	}
	raw, err := s.sessions.Create(ctx, credential.UserID, SessionMetadata{})
	if err != nil {
		return AuthSession{}, err
	}
	return s.authSession(raw.Token, raw.ExpiresAt, credential.MustChangePassword), nil
}

func (s *AuthService) recordInvalidLogin(ctx context.Context, email, clientIP string) error {
	if s.limiter != nil {
		if err := s.limiter.RecordFailure(ctx, email, clientIP); err != nil {
			return err
		}
	}
	return ErrInvalidCredentials
}

func (s *AuthService) Restore(ctx context.Context, rawToken string) (AuthSession, error) {
	if len(s.csrfSecret) == 0 {
		return AuthSession{}, errors.New("CSRF secret is not configured")
	}
	session, err := s.sessions.Authenticate(ctx, rawToken)
	if err != nil {
		return AuthSession{}, err
	}
	store, err := s.passwordStore()
	if err != nil {
		return AuthSession{}, err
	}
	credential, err := store.FindByUserID(ctx, session.UserID)
	if errors.Is(err, ErrCredentialNotFound) {
		return AuthSession{}, ErrSessionInvalid
	}
	if err != nil {
		return AuthSession{}, err
	}
	if credential == nil || !credential.Active {
		return AuthSession{}, ErrSessionInvalid
	}
	return s.authSession(rawToken, session.ExpiresAt, credential.MustChangePassword), nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	session, err := s.sessions.Authenticate(ctx, rawToken)
	if err != nil {
		return err
	}
	return s.sessions.Revoke(ctx, session.ID)
}

func (s *AuthService) ChangePassword(ctx context.Context, rawToken, currentPassword, newPassword string) (AuthSession, error) {
	if len(s.csrfSecret) == 0 {
		return AuthSession{}, errors.New("CSRF secret is not configured")
	}
	session, err := s.sessions.Authenticate(ctx, rawToken)
	if err != nil {
		return AuthSession{}, err
	}
	store, err := s.passwordStore()
	if err != nil {
		return AuthSession{}, err
	}
	credential, err := store.FindByUserID(ctx, session.UserID)
	if errors.Is(err, ErrCredentialNotFound) {
		return AuthSession{}, ErrSessionInvalid
	}
	if err != nil {
		return AuthSession{}, err
	}
	if credential == nil || !credential.Active {
		return AuthSession{}, ErrSessionInvalid
	}
	valid, err := s.hasher.Verify(credential.PasswordHash, currentPassword)
	if err != nil || !valid {
		return AuthSession{}, ErrPasswordInvalid
	}
	if len(newPassword) < 12 || len(newPassword) > 1<<20 {
		return AuthSession{}, ErrPasswordPolicy
	}
	reused, err := s.hasher.Verify(credential.PasswordHash, newPassword)
	if err != nil {
		return AuthSession{}, err
	}
	if reused {
		return AuthSession{}, ErrPasswordReuse
	}
	newHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return AuthSession{}, err
	}
	if err := store.ChangePasswordAndRevokeSessions(ctx, session.UserID, newHash, s.sessions.clock().UTC()); err != nil {
		return AuthSession{}, err
	}
	raw, err := s.sessions.Create(ctx, session.UserID, SessionMetadata{})
	if err != nil {
		return AuthSession{}, err
	}
	return s.authSession(raw.Token, raw.ExpiresAt, false), nil
}

func (s *AuthService) passwordStore() (PasswordCredentialRepository, error) {
	store, ok := s.credentials.(PasswordCredentialRepository)
	if !ok {
		return nil, errors.New("credential repository does not support session restoration or password change")
	}
	return store, nil
}

func (s *AuthService) authSession(rawToken string, expiresAt time.Time, mustChange bool) AuthSession {
	return AuthSession{
		Token: rawToken, CSRFToken: security.DeriveCSRFToken(s.csrfSecret, rawToken),
		ExpiresAt: expiresAt.UTC(), MustChangePassword: mustChange,
	}
}
