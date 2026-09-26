package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// These tests describe the login ports before service.go exists. In particular,
// the membership port returns a count so auth does not depend on workshop types.
func TestLoginReturnsOneGenericErrorForUnknownInactiveAndWrongPassword(t *testing.T) {
	hasher := NewPasswordHasher(DefaultPasswordParams())
	hash, err := hasher.Hash("Correct horse battery staple 7!")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name        string
		email       string
		password    string
		credential  *Credential
		memberships int
	}{
		{name: "unknown account", email: "missing@example.com", password: "wrong"},
		{name: "inactive account", email: "owner@example.com", password: "Correct horse battery staple 7!", credential: &Credential{UserID: uuid.New(), PasswordHash: hash, Active: false}, memberships: 1},
		{name: "wrong password", email: "owner@example.com", password: "wrong", credential: &Credential{UserID: uuid.New(), PasswordHash: hash, Active: true}, memberships: 1},
		{name: "no active membership", email: "owner@example.com", password: "Correct horse battery staple 7!", credential: &Credential{UserID: uuid.New(), PasswordHash: hash, Active: true}},
		{name: "ambiguous active membership", email: "owner@example.com", password: "Correct horse battery staple 7!", credential: &Credential{UserID: uuid.New(), PasswordHash: hash, Active: true}, memberships: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessions := &fakeSessionRepository{}
			service := NewAuthService(
				&loginCredentialRepo{credential: tc.credential},
				&loginMemberships{count: tc.memberships},
				NewSessionService(sessions, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32))),
				hasher,
				&loginLimiter{attempts: make(map[string]int)},
			)

			_, err := service.Login(context.Background(), tc.email, tc.password, "192.0.2.10")
			if !errors.Is(err, ErrInvalidCredentials) {
				t.Fatalf("Login() error = %v, want generic ErrInvalidCredentials", err)
			}
			if sessions.inserted != nil {
				t.Fatal("invalid login created a session")
			}
		})
	}
}

func TestLoginRateLimitsUnknownAccountsAfterFiveFailures(t *testing.T) {
	service := NewAuthService(
		&loginCredentialRepo{},
		&loginMemberships{count: 1},
		NewSessionService(&fakeSessionRepository{}, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32))),
		NewPasswordHasher(DefaultPasswordParams()),
		&loginLimiter{attempts: make(map[string]int)},
	)

	for i := 0; i < 5; i++ {
		_, err := service.Login(context.Background(), "missing@example.com", "wrong", "192.0.2.10")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d error = %v, want ErrInvalidCredentials", i+1, err)
		}
	}
	_, err := service.Login(context.Background(), "missing@example.com", "wrong", "192.0.2.10")
	if !errors.Is(err, ErrTooManyAttempts) {
		t.Fatalf("sixth attempt error = %v, want ErrTooManyAttempts", err)
	}
}

func TestChangePasswordTransactionFailurePreservesOldCredentialAndSession(t *testing.T) {
	const oldPassword = "Temporary secure passphrase 9!"
	const newPassword = "New owner passphrase 10!"
	const rawToken = "3fUHx2rXJ0l6R1l-VV5I8JH8sRj2r0N-fzpZ08O1ZJc"
	hasher := NewPasswordHasher(DefaultPasswordParams())
	oldHash, err := hasher.Hash(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	transactionFailure := errors.New("revoke sessions failed inside transaction")
	credentials := &failingPasswordCredentialRepo{
		credential: Credential{UserID: testUserID, PasswordHash: oldHash, Active: true, MustChangePassword: true},
	}
	sessionRepo := &fakeSessionRepository{found: Session{
		ID: testSessionID, UserID: testUserID, ExpiresAt: testNow.Add(time.Hour),
	}, changePasswordAndInsertErr: transactionFailure}
	service := NewAuthService(credentials, &loginMemberships{count: 1},
		NewSessionService(sessionRepo, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32))),
		hasher, &loginLimiter{attempts: make(map[string]int)})

	_, err = service.ChangePassword(context.Background(), rawToken, oldPassword, newPassword)
	if !errors.Is(err, transactionFailure) {
		t.Fatalf("ChangePassword() error = %v, want transaction failure", err)
	}
	oldMatches, err := hasher.Verify(credentials.credential.PasswordHash, oldPassword)
	if err != nil || !oldMatches || !credentials.credential.MustChangePassword {
		t.Fatal("failed transaction changed the credential or cleared the mandatory-change flag")
	}
	if sessionRepo.inserted != nil {
		t.Fatal("failed transaction created a replacement session")
	}
	restored, err := service.Restore(context.Background(), rawToken)
	if err != nil || !restored.MustChangePassword {
		t.Fatalf("old session was lost after rolled-back password change: session = %+v, error = %v", restored, err)
	}
}

func TestLoginRejectsCredentialChangedAfterVerification(t *testing.T) {
	const password = "Correct horse battery staple 7!"
	hasher := NewPasswordHasher(DefaultPasswordParams())
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatal(err)
	}
	sessions := &fakeSessionRepository{insertErr: ErrCredentialChanged, insertForCredentialErr: ErrCredentialChanged}
	service := NewAuthService(
		&loginCredentialRepo{credential: &Credential{UserID: testUserID, PasswordHash: hash, Active: true}},
		&loginMemberships{count: 1},
		NewSessionService(sessions, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32))),
		hasher,
		&loginLimiter{attempts: make(map[string]int)},
	)

	_, err = service.Login(context.Background(), "owner@example.com", password, "192.0.2.10")

	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want generic ErrInvalidCredentials", err)
	}
	if sessions.inserted != nil && sessions.insertForCredentialHash == "" {
		t.Fatal("login used a non-atomic session insert")
	}
}

func TestPasswordChangeRevokesAndCreatesSessionAtomically(t *testing.T) {
	const oldPassword = "Temporary secure passphrase 9!"
	const newPassword = "New owner passphrase 10!"
	const rawToken = "3fUHx2rXJ0l6R1l-VV5I8JH8sRj2r0N-fzpZ08O1ZJc"
	hasher := NewPasswordHasher(DefaultPasswordParams())
	oldHash, err := hasher.Hash(oldPassword)
	if err != nil {
		t.Fatal(err)
	}
	credentials := &failingPasswordCredentialRepo{credential: Credential{UserID: testUserID, PasswordHash: oldHash, Active: true, MustChangePassword: true}}
	sessions := &fakeSessionRepository{found: Session{ID: testSessionID, UserID: testUserID, ExpiresAt: testNow.Add(time.Hour)}}
	service := NewAuthService(credentials, &loginMemberships{count: 1},
		NewSessionService(sessions, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32))),
		hasher, &loginLimiter{attempts: make(map[string]int)})

	rotated, err := service.ChangePassword(context.Background(), rawToken, oldPassword, newPassword)

	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if sessions.passwordExpectedHash != oldHash || sessions.passwordReplacementHash == "" {
		t.Fatal("password change did not use the atomic credential/session operation")
	}
	if sessions.inserted == nil || sessions.inserted.UserID != testUserID || rotated.Token == "" {
		t.Fatal("password change did not atomically prepare the replacement session")
	}
}

type loginCredentialRepo struct{ credential *Credential }

func (r *loginCredentialRepo) FindByEmail(_ context.Context, _ string) (*Credential, error) {
	return r.credential, nil
}

type loginMemberships struct{ count int }

func (m *loginMemberships) ResolveActive(_ context.Context, _ uuid.UUID) (ActiveMembership, error) {
	if m.count != 1 {
		return ActiveMembership{}, ErrInvalidCredentials
	}
	return ActiveMembership{WorkshopID: uuid.New(), Role: "OWNER"}, nil
}

// A stateful port fixture gives the service a real boundary: failures must be
// recorded even when no account exists, or its sixth call will not be blocked.
type loginLimiter struct{ attempts map[string]int }

func (l *loginLimiter) Allow(_ context.Context, email, ip string) (bool, error) {
	return l.attempts[email+"|"+ip] < 5, nil
}

func (l *loginLimiter) RecordFailure(_ context.Context, email, ip string) error {
	l.attempts[email+"|"+ip]++
	return nil
}

type failingPasswordCredentialRepo struct {
	credential Credential
}

func (r *failingPasswordCredentialRepo) FindByEmail(_ context.Context, _ string) (*Credential, error) {
	return &r.credential, nil
}

func (r *failingPasswordCredentialRepo) FindByUserID(_ context.Context, _ uuid.UUID) (*Credential, error) {
	return &r.credential, nil
}
