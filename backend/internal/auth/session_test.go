package auth

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/google/uuid"
)

var (
	testNow       = time.Date(2026, time.September, 22, 15, 0, 0, 0, time.UTC)
	testUserID    = uuid.MustParse("f995e452-5158-4a27-b76d-918aa004811f")
	testSessionID = uuid.MustParse("87a68362-7676-4f20-a16a-c100309cb9f9")
	testPepper    = []byte("test-only-session-pepper")
)

func TestSessionCreateStoresOnlyPepperedTokenDigest(t *testing.T) {
	repo := &fakeSessionRepository{}
	randomBytes := make([]byte, 32)
	for i := range randomBytes {
		randomBytes[i] = byte(i + 1)
	}
	service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(randomBytes))

	raw, err := service.Create(context.Background(), testUserID, SessionMetadata{})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.inserted == nil {
		t.Fatal("Create() did not insert a session")
	}

	wantToken := base64.RawURLEncoding.EncodeToString(randomBytes)
	if raw.Token != wantToken {
		t.Fatalf("Create() token = %q, want base64url encoding of 32 random bytes", raw.Token)
	}
	if len(raw.Token) != 43 {
		t.Fatalf("Create() token length = %d, want 43", len(raw.Token))
	}

	wantDigest := sessionDigest(testPepper, raw.Token)
	if repo.inserted.TokenHash != wantDigest {
		t.Fatalf("inserted token hash = %x, want HMAC-SHA-256 digest %x", repo.inserted.TokenHash, wantDigest)
	}
	wantCSRFTokenHash := sha256.Sum256([]byte(security.DeriveCSRFToken(testPepper, raw.Token)))
	if repo.inserted.CSRFTokenHash != wantCSRFTokenHash {
		t.Fatalf("inserted CSRF token hash = %x, want SHA-256 digest %x", repo.inserted.CSRFTokenHash, wantCSRFTokenHash)
	}
	if bytes.Equal(repo.inserted.TokenHash[:], []byte(raw.Token)) {
		t.Fatal("repository received the raw session token instead of its digest")
	}
	if repo.inserted.UserID != testUserID {
		t.Fatalf("inserted user ID = %s, want %s", repo.inserted.UserID, testUserID)
	}
	if want := testNow.Add(8 * time.Hour); !repo.inserted.ExpiresAt.Equal(want) {
		t.Fatalf("inserted expiry = %s, want %s", repo.inserted.ExpiresAt, want)
	}
}

func TestSessionPrepareReturnsRawBrowserValuesWithoutWriting(t *testing.T) {
	repo := &fakeSessionRepository{}
	randomBytes := make([]byte, 32)
	for i := range randomBytes {
		randomBytes[i] = byte(i + 1)
	}
	service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(randomBytes))

	prepared, err := service.Prepare(testUserID, SessionMetadata{IPPrefix: "203.0.113.0/24", UserAgent: "browser"})

	if err != nil {
		t.Fatalf("Prepare() error = %v", err)
	}
	if repo.inserted != nil {
		t.Fatal("Prepare() wrote a session before its caller's transaction")
	}
	if prepared.Token == "" || prepared.CSRFToken == "" || prepared.Created.UserID != testUserID {
		t.Fatalf("Prepare() returned incomplete values: %+v", prepared)
	}
	if prepared.CSRFToken != security.DeriveCSRFToken(testPepper, prepared.Token) {
		t.Fatal("Prepare() returned a CSRF token not bound to the opaque token")
	}
	if prepared.Created.Metadata.IPPrefix != "203.0.113.0/24" || prepared.Created.Metadata.UserAgent != "browser" {
		t.Fatalf("Prepare() lost session metadata: %+v", prepared.Created.Metadata)
	}
}

func TestSessionAuthenticateUsesPepperedDigest(t *testing.T) {
	const rawToken = "3fUHx2rXJ0l6R1l-VV5I8JH8sRj2r0N-fzpZ08O1ZJc"
	wantSession := Session{
		ID:        testSessionID,
		UserID:    testUserID,
		ExpiresAt: testNow.Add(time.Hour),
	}
	repo := &fakeSessionRepository{found: wantSession}
	service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32)))

	got, err := service.Authenticate(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	if got.ID != wantSession.ID || got.UserID != wantSession.UserID {
		t.Fatalf("Authenticate() = %+v, want session ID %s for user %s", got, wantSession.ID, wantSession.UserID)
	}
	if wantDigest := sessionDigest(testPepper, rawToken); repo.findHash != wantDigest {
		t.Fatalf("repository lookup digest = %x, want %x", repo.findHash, wantDigest)
	}
	if !repo.findNow.Equal(testNow) {
		t.Fatalf("repository lookup time = %s, want %s", repo.findNow, testNow)
	}
}

func TestSessionAuthenticateRejectsRevokedSession(t *testing.T) {
	revokedAt := testNow.Add(-time.Minute)
	repo := &fakeSessionRepository{found: Session{
		ID:        testSessionID,
		UserID:    testUserID,
		ExpiresAt: testNow.Add(time.Hour),
		RevokedAt: &revokedAt,
	}}
	service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32)))

	if _, err := service.Authenticate(context.Background(), "revoked-token"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Authenticate() error = %v, want ErrSessionInvalid for revoked session", err)
	}
}

func TestSessionAuthenticateRejectsExpiredSession(t *testing.T) {
	repo := &fakeSessionRepository{found: Session{
		ID:        testSessionID,
		UserID:    testUserID,
		ExpiresAt: testNow,
	}}
	service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32)))

	if _, err := service.Authenticate(context.Background(), "expired-token"); !errors.Is(err, ErrSessionInvalid) {
		t.Fatalf("Authenticate() error = %v, want ErrSessionInvalid for expired session", err)
	}
}

func TestSessionAuthenticatePreservesRepositoryErrors(t *testing.T) {
	operationalErr := errors.New("database unavailable")
	tests := []struct {
		name    string
		repoErr error
		wantErr error
	}{
		{name: "missing active session", repoErr: ErrSessionInvalid, wantErr: ErrSessionInvalid},
		{name: "operational failure", repoErr: operationalErr, wantErr: operationalErr},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeSessionRepository{findErr: tc.repoErr}
			service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32)))

			if _, err := service.Authenticate(context.Background(), "opaque-token"); !errors.Is(err, tc.wantErr) {
				t.Fatalf("Authenticate() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestSessionRevokePassesSessionAndCurrentTimeToRepository(t *testing.T) {
	repo := &fakeSessionRepository{}
	service := NewSessionService(repo, testPepper, fixedClock(testNow), bytes.NewReader(make([]byte, 32)))

	if err := service.Revoke(context.Background(), testSessionID); err != nil {
		t.Fatalf("Revoke() error = %v", err)
	}
	if repo.revokedID != testSessionID {
		t.Fatalf("Revoke() repository ID = %s, want %s", repo.revokedID, testSessionID)
	}
	if !repo.revokedAt.Equal(testNow) {
		t.Fatalf("Revoke() repository time = %s, want %s", repo.revokedAt, testNow)
	}
}

func fixedClock(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

func sessionDigest(pepper []byte, rawToken string) [32]byte {
	mac := hmac.New(sha256.New, pepper)
	_, _ = mac.Write([]byte(rawToken))
	var digest [32]byte
	copy(digest[:], mac.Sum(nil))
	return digest
}

type fakeSessionRepository struct {
	inserted                   *NewSession
	insertErr                  error
	insertForCredentialHash    string
	insertForCredentialErr     error
	passwordExpectedHash       string
	passwordReplacementHash    string
	passwordChangedAt          time.Time
	changePasswordAndInsertErr error
	found                      Session
	findErr                    error
	findHash                   [32]byte
	findNow                    time.Time
	revokedID                  uuid.UUID
	revokedAt                  time.Time
}

func (r *fakeSessionRepository) Insert(_ context.Context, in NewSession) (Session, error) {
	r.inserted = &in
	if r.insertErr != nil {
		return Session{}, r.insertErr
	}
	return Session{
		ID:        testSessionID,
		UserID:    in.UserID,
		ExpiresAt: in.ExpiresAt,
	}, nil
}

func (r *fakeSessionRepository) InsertForCredential(_ context.Context, verifiedHash string, in NewSession) (Session, error) {
	r.insertForCredentialHash = verifiedHash
	if r.insertForCredentialErr != nil {
		return Session{}, r.insertForCredentialErr
	}
	r.inserted = &in
	return Session{ID: testSessionID, UserID: in.UserID, ExpiresAt: in.ExpiresAt}, nil
}

func (r *fakeSessionRepository) ChangePasswordAndInsert(_ context.Context, _ uuid.UUID, expectedHash, replacementHash string, changedAt time.Time, in NewSession) (Session, error) {
	r.passwordExpectedHash = expectedHash
	r.passwordReplacementHash = replacementHash
	r.passwordChangedAt = changedAt
	if r.changePasswordAndInsertErr != nil {
		return Session{}, r.changePasswordAndInsertErr
	}
	r.inserted = &in
	return Session{ID: testSessionID, UserID: in.UserID, ExpiresAt: in.ExpiresAt}, nil
}

func (r *fakeSessionRepository) FindActiveByTokenHash(_ context.Context, hash [32]byte, now time.Time) (Session, error) {
	r.findHash = hash
	r.findNow = now
	return r.found, r.findErr
}

func (r *fakeSessionRepository) Revoke(_ context.Context, sessionID uuid.UUID, now time.Time) error {
	r.revokedID = sessionID
	r.revokedAt = now
	return nil
}

func (r *fakeSessionRepository) RevokeAllForUser(_ context.Context, _ uuid.UUID, _ time.Time) error {
	return nil
}
