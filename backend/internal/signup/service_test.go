package signup

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	platformmail "github.com/Josed20/Tallerflow/backend/platform/mail"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeRepository struct {
	registered    []PendingAccount
	registerIssue bool
	reissueEmail  string
	reissueToken  []byte
	reissueIssue  bool
	consumed      [][]byte
	consumeUser   uuid.UUID
	consumeErr    error
}

func (r *fakeRepository) Register(_ context.Context, account PendingAccount) (bool, error) {
	r.registered = append(r.registered, account)
	return r.registerIssue, nil
}

func (r *fakeRepository) ReissueToken(_ context.Context, email string, tokenHash []byte, _ time.Time) (Recipient, bool, error) {
	r.reissueEmail, r.reissueToken = email, tokenHash
	return Recipient{Name: "Lucho"}, r.reissueIssue, nil
}

func (r *fakeRepository) ConsumeToken(_ context.Context, tokenHash []byte) (uuid.UUID, error) {
	r.consumed = append(r.consumed, tokenHash)
	return r.consumeUser, r.consumeErr
}

type fakeHasher struct{ calls int }

func (h *fakeHasher) Hash(password string) (string, error) {
	h.calls++
	return "hashed:" + password, nil
}

type fakeSessions struct {
	userID   uuid.UUID
	metadata auth.SessionMetadata
}

func (s *fakeSessions) Create(_ context.Context, userID uuid.UUID, metadata auth.SessionMetadata) (auth.RawSession, error) {
	s.userID, s.metadata = userID, metadata
	return auth.RawSession{Token: "session-token", CSRFToken: "csrf-token", ExpiresAt: time.Date(2026, 10, 22, 20, 0, 0, 0, time.UTC)}, nil
}

func newTestService(t *testing.T, repo *fakeRepository, enabled bool) (*Service, *fakeHasher, *fakeSessions, *platformmail.Fake) {
	t.Helper()
	hasher, sessions, mailer := &fakeHasher{}, &fakeSessions{}, platformmail.NewFake()
	service, err := NewService(repo, hasher, sessions, mailer, Config{
		Enabled: enabled, BaseURL: "https://tallerflowpe.com/", MinimumDuration: time.Millisecond,
	}, func() time.Time { return time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC) })
	require.NoError(t, err)
	service.dispatch = func(send func()) { send() }
	return service, hasher, sessions, mailer
}

func validInput() Input {
	return Input{WorkshopName: " Confecciones Lucho ", OwnerName: " Lucho Quispe ", Email: " Lucho@Taller.PE ", Password: "Secure signup password 2026!", AcceptTerms: true}
}

func TestRegisterIsRefusedWhileSignupIsClosed(t *testing.T) {
	repo := &fakeRepository{}
	service, hasher, _, mailer := newTestService(t, repo, false)

	err := service.Register(context.Background(), validInput())

	require.ErrorIs(t, err, ErrClosed)
	require.Empty(t, repo.registered)
	require.Zero(t, hasher.calls)
	require.Empty(t, mailer.Messages())
}

func TestRegisterReportsEveryInvalidFieldIncludingTerms(t *testing.T) {
	service, _, _, _ := newTestService(t, &fakeRepository{}, true)

	err := service.Register(context.Background(), Input{WorkshopName: " ", Email: "not-an-email", Password: "short"})

	var validation *ValidationError
	require.ErrorAs(t, err, &validation)
	require.Equal(t, []string{"workshop_name", "owner_name", "email", "password", "accept_terms"}, validation.Fields)
}

func TestRegisterStoresOnlyTheTokenDigestAndEmailsTheLink(t *testing.T) {
	repo := &fakeRepository{registerIssue: true}
	service, _, _, mailer := newTestService(t, repo, true)

	require.NoError(t, service.Register(context.Background(), validInput()))

	require.Len(t, repo.registered, 1)
	account := repo.registered[0]
	require.Equal(t, "lucho@taller.pe", account.Email)
	require.Equal(t, "Lucho Quispe", account.OwnerName)
	require.Equal(t, "Confecciones Lucho", account.WorkshopName)
	require.Equal(t, "hashed:Secure signup password 2026!", account.PasswordHash)
	require.Equal(t, time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC), account.ExpiresAt)

	message, ok := mailer.LastMessage()
	require.True(t, ok)
	require.Equal(t, "lucho@taller.pe", message.To)
	link := linkIn(t, message.Text)
	require.True(t, strings.HasPrefix(link, "https://tallerflowpe.com/verificar-correo?token="), link)
	raw := tokenFrom(t, link)
	require.Len(t, raw, tokenLength)
	digest := sha256.Sum256([]byte(raw))
	require.Equal(t, digest[:], account.TokenHash, "only the SHA-256 of the emailed token is stored")
	require.NotContains(t, string(account.TokenHash), raw)
	require.Contains(t, message.Text, "Confecciones Lucho")
}

func TestRegisterForAVerifiedAddressAnswersTheSameAndSendsNothing(t *testing.T) {
	repo := &fakeRepository{registerIssue: false}
	service, hasher, _, mailer := newTestService(t, repo, true)

	require.NoError(t, service.Register(context.Background(), validInput()))

	require.Equal(t, 1, hasher.calls, "the password is hashed even when nothing is created")
	require.Empty(t, mailer.Messages())
}

func TestResendOnlyEmailsPendingAccounts(t *testing.T) {
	repo := &fakeRepository{reissueIssue: false}
	service, _, _, mailer := newTestService(t, repo, true)
	require.NoError(t, service.Resend(context.Background(), " Unknown@Example.com "))
	require.Equal(t, "unknown@example.com", repo.reissueEmail)
	require.Empty(t, mailer.Messages())

	repo.reissueIssue = true
	require.NoError(t, service.Resend(context.Background(), "lucho@taller.pe"))
	message, ok := mailer.LastMessage()
	require.True(t, ok)
	digest := sha256.Sum256([]byte(tokenFrom(t, linkIn(t, message.Text))))
	require.Equal(t, digest[:], repo.reissueToken)

	var validation *ValidationError
	require.ErrorAs(t, service.Resend(context.Background(), "nope"), &validation)
}

func TestVerifyRejectsMalformedTokensBeforeTheDatabase(t *testing.T) {
	repo := &fakeRepository{}
	service, _, _, _ := newTestService(t, repo, true)

	for _, raw := range []string{"", "short", strings.Repeat("a", 42), strings.Repeat("a", 44), strings.Repeat("a", 42) + "="} {
		_, err := service.Verify(context.Background(), raw, auth.SessionMetadata{})
		require.ErrorIs(t, err, ErrTokenInvalid, raw)
	}
	require.Empty(t, repo.consumed)
}

func TestVerifyOpensABrowserSessionForTheOwner(t *testing.T) {
	userID := uuid.New()
	repo := &fakeRepository{consumeUser: userID}
	service, _, sessions, _ := newTestService(t, repo, true)
	raw, digest, err := GenerateToken()
	require.NoError(t, err)

	session, err := service.Verify(context.Background(), raw, auth.SessionMetadata{IPPrefix: "203.0.113.0/24", Kind: auth.SessionKindMobile})

	require.NoError(t, err)
	require.Equal(t, "session-token", session.Token)
	require.Equal(t, [][]byte{digest}, repo.consumed)
	require.Equal(t, userID, sessions.userID)
	require.Equal(t, auth.SessionKindWeb, sessions.metadata.Kind, "the emailed link always opens a browser session")
}

func TestVerifyPassesThroughInvalidTokens(t *testing.T) {
	repo := &fakeRepository{consumeErr: ErrTokenInvalid}
	service, _, sessions, _ := newTestService(t, repo, true)
	raw, _, err := GenerateToken()
	require.NoError(t, err)

	_, err = service.Verify(context.Background(), raw, auth.SessionMetadata{})

	require.True(t, errors.Is(err, ErrTokenInvalid))
	require.Equal(t, uuid.Nil, sessions.userID)
}

func linkIn(t *testing.T, text string) string {
	t.Helper()
	for _, field := range strings.Fields(text) {
		if strings.Contains(field, "/verificar-correo?token=") {
			return field
		}
	}
	t.Fatalf("no verification link in %q", text)
	return ""
}

func tokenFrom(t *testing.T, link string) string {
	t.Helper()
	parsed, err := url.Parse(link)
	require.NoError(t, err)
	return parsed.Query().Get("token")
}
