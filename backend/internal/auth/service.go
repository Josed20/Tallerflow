package auth

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/httpx"
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
	// ErrEmailNotVerified is returned only after the password matched, so it
	// never tells a stranger that an address is registered.
	ErrEmailNotVerified = errors.New("email is not verified")
	// ErrWorkshopSuspended keeps members of a suspended or deleting workshop
	// out. Like ErrEmailNotVerified, login returns it only after the password.
	ErrWorkshopSuspended = errors.New("workshop is suspended")
)

var nonexistentCredentialHash struct {
	sync.Once
	value string
	err   error
}

type Credential struct {
	UserID             uuid.UUID
	Email              string
	DisplayName        string
	PasswordHash       string
	Active             bool
	MustChangePassword bool
	// PendingEmailVerification is true for an open-signup owner who has not
	// opened the emailed link yet. The zero value is a verified account.
	PendingEmailVerification bool
}

type CredentialRepository interface {
	FindByEmail(context.Context, string) (*Credential, error)
}

// PasswordCredentialRepository is needed by restoration and password changes.
// It is separate so login-only adapters can stay small.
type PasswordCredentialRepository interface {
	CredentialRepository
	FindByUserID(context.Context, uuid.UUID) (*Credential, error)
}

type MembershipResolver interface {
	ResolveActive(context.Context, uuid.UUID) (ActiveMembership, error)
}

// WorkshopGate reports whether the member's workshop may be used. Login and
// every private route check it, so a suspension from the platform console
// stops web and mobile sessions alike, not only the ones it revoked. Logout
// and the password change skip it, like the bootstrap password gate.
type WorkshopGate interface {
	EnsureActive(ctx context.Context, workshopID uuid.UUID) error
}

type LoginLimiter interface {
	Allow(context.Context, string, string) (bool, error)
	RecordFailure(context.Context, string, string) error
}

type PasswordHashing interface {
	Hash(string) (string, error)
	Verify(string, string) (bool, error)
}

type AuthSession struct {
	Token              string
	CSRFToken          string
	Kind               SessionKind
	ExpiresAt          time.Time
	MustChangePassword bool
	Principal          httpx.Principal
}

type AuthService struct {
	credentials CredentialRepository
	memberships MembershipResolver
	sessions    *SessionService
	hasher      PasswordHashing
	limiter     LoginLimiter
	workshops   WorkshopGate
	csrfSecret  []byte
}

// UseWorkshopGate turns on the workshop status check. Without a gate, as in
// unit tests, every resolved membership is treated as active.
func (s *AuthService) UseWorkshopGate(gate WorkshopGate) { s.workshops = gate }

// EnsureWorkshopActive returns ErrWorkshopSuspended when the workshop of an
// authenticated principal may not be used.
func (s *AuthService) EnsureWorkshopActive(ctx context.Context, workshopID uuid.UUID) error {
	return s.ensureWorkshopActive(ctx, workshopID)
}

func (s *AuthService) ensureWorkshopActive(ctx context.Context, workshopID uuid.UUID) error {
	if s.workshops == nil {
		return nil
	}
	return s.workshops.EnsureActive(ctx, workshopID)
}

// A purpose label in DeriveCSRFToken separates CSRF values from session-token
// digests even though both use the server's session pepper.
func NewAuthService(credentials CredentialRepository, memberships MembershipResolver, sessions *SessionService, hasher PasswordHashing, limiter LoginLimiter) *AuthService {
	service := &AuthService{credentials: credentials, memberships: memberships, sessions: sessions, hasher: hasher, limiter: limiter}
	if sessions != nil {
		service.csrfSecret = append([]byte(nil), sessions.pepper...)
	}
	return service
}

func (s *AuthService) Login(ctx context.Context, email, password, clientIP string, kind SessionKind) (AuthSession, error) {
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
	if credential.PendingEmailVerification {
		return AuthSession{}, ErrEmailNotVerified
	}
	membership, err := s.memberships.ResolveActive(ctx, credential.UserID)
	if err != nil {
		if !errors.Is(err, ErrInvalidCredentials) {
			return AuthSession{}, err
		}
		return AuthSession{}, s.recordInvalidLogin(ctx, email, clientIP)
	}
	if err := s.ensureWorkshopActive(ctx, membership.WorkshopID); err != nil {
		return AuthSession{}, err
	}
	if len(s.csrfSecret) == 0 {
		return AuthSession{}, errors.New("CSRF secret is not configured")
	}
	raw, err := s.sessions.CreateForCredential(ctx, credential.UserID, credential.PasswordHash, SessionMetadata{Kind: kind})
	if err != nil {
		if errors.Is(err, ErrCredentialChanged) {
			return AuthSession{}, s.recordInvalidLogin(ctx, email, clientIP)
		}
		return AuthSession{}, err
	}
	return s.authSession(raw.Token, kind, raw.ExpiresAt, s.principal(credential, membership), credential.MustChangePassword), nil
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
	membership, err := s.memberships.ResolveActive(ctx, credential.UserID)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return AuthSession{}, ErrSessionInvalid
		}
		return AuthSession{}, err
	}
	return s.authSession(rawToken, session.Kind, session.ExpiresAt, s.principal(credential, membership), credential.MustChangePassword), nil
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
	membership, err := s.memberships.ResolveActive(ctx, credential.UserID)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return AuthSession{}, ErrSessionInvalid
		}
		return AuthSession{}, err
	}
	newHash, err := s.hasher.Hash(newPassword)
	if err != nil {
		return AuthSession{}, err
	}
	// The replacement keeps the transport of the session that asked for it;
	// every other session of the user, web or mobile, is revoked.
	raw, err := s.sessions.ChangePasswordAndCreate(ctx, session.UserID, credential.PasswordHash, newHash, SessionMetadata{Kind: session.Kind})
	if err != nil {
		if errors.Is(err, ErrCredentialChanged) {
			return AuthSession{}, ErrPasswordInvalid
		}
		return AuthSession{}, err
	}
	return s.authSession(raw.Token, session.Kind, raw.ExpiresAt, s.principal(credential, membership), false), nil
}

func (s *AuthService) passwordStore() (PasswordCredentialRepository, error) {
	store, ok := s.credentials.(PasswordCredentialRepository)
	if !ok {
		return nil, errors.New("credential repository does not support session restoration or password change")
	}
	return store, nil
}

// authSession derives the CSRF token only for browser sessions: bearer
// sessions are not sent automatically by the browser and need none.
func (s *AuthService) authSession(rawToken string, kind SessionKind, expiresAt time.Time, principal httpx.Principal, mustChange bool) AuthSession {
	principal.PasswordChangeRequired = mustChange
	kind = kind.normalized()
	csrfToken := ""
	if kind == SessionKindWeb {
		csrfToken = security.DeriveCSRFToken(s.csrfSecret, rawToken)
	}
	return AuthSession{
		Token: rawToken, CSRFToken: csrfToken, Kind: kind,
		ExpiresAt: expiresAt.UTC(), MustChangePassword: mustChange, Principal: principal,
	}
}

func (s *AuthService) principal(credential *Credential, membership ActiveMembership) httpx.Principal {
	return httpx.Principal{
		UserID: credential.UserID, Email: credential.Email, DisplayName: credential.DisplayName,
		WorkshopID: membership.WorkshopID, Role: membership.Role,
	}
}
