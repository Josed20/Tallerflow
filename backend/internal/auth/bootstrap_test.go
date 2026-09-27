package auth

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/platform/security"
	"github.com/google/uuid"
)

func TestBootstrapCreatesOwnerWithTemporaryCredentialAndAudit(t *testing.T) {
	store := newBootstrapStoreFake()
	service := NewBootstrapService(store, NewPasswordHasher(DefaultPasswordParams()))
	input := BootstrapInput{
		Email: "owner@tallerflow.pe", Name: "Owner Demo",
		WorkshopName: "Taller Demo", Password: "Temporary secure passphrase 9!",
	}

	result, err := service.CreateOwner(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.UserID == uuid.Nil || result.WorkshopID == uuid.Nil || result.Role != "OWNER" || !result.MustChangePassword {
		t.Fatalf("unexpected bootstrap result: %+v", result)
	}
	if store.state.users[input.Email] != result.UserID {
		t.Fatal("user was not created")
	}
	credential := store.state.credentials[result.UserID]
	if credential.passwordHash == "" || credential.passwordHash == input.Password || !credential.mustChange {
		t.Fatal("bootstrap did not persist a hashed, temporary credential")
	}
	ok, err := NewPasswordHasher(DefaultPasswordParams()).Verify(credential.passwordHash, input.Password)
	if err != nil || !ok {
		t.Fatalf("bootstrap credential cannot verify: ok=%t err=%v", ok, err)
	}
	if store.state.workshops[result.WorkshopID] != "Taller Demo|America/Lima" {
		t.Fatal("workshop or timezone was not created")
	}
	if len(store.state.memberships) != 1 || store.state.memberships[0] != (bootstrapMembership{result.UserID, result.WorkshopID, "OWNER"}) {
		t.Fatal("exactly one OWNER membership was not created")
	}
	if len(store.state.audit) != 1 || store.state.audit[0] != "OWNER_BOOTSTRAPPED" {
		t.Fatal("bootstrap was not audited")
	}
}

func TestBootstrapRollsBackEverythingWhenMembershipFails(t *testing.T) {
	store := newBootstrapStoreFake()
	store.failMembership = true
	service := NewBootstrapService(store, NewPasswordHasher(DefaultPasswordParams()))
	_, err := service.CreateOwner(context.Background(), BootstrapInput{
		Email: "owner@tallerflow.pe", Name: "Owner Demo",
		WorkshopName: "Taller Demo", Password: "Temporary secure passphrase 9!",
	})
	if err == nil {
		t.Fatal("bootstrap succeeded despite a membership insertion failure")
	}
	if len(store.state.users) != 0 || len(store.state.credentials) != 0 || len(store.state.workshops) != 0 || len(store.state.memberships) != 0 || len(store.state.audit) != 0 {
		t.Fatalf("bootstrap left partial state after rollback: %+v", store.state)
	}
}

func TestBootstrapDoesNotChangeExistingOwner(t *testing.T) {
	store := newBootstrapStoreFake()
	service := NewBootstrapService(store, NewPasswordHasher(DefaultPasswordParams()))
	input := BootstrapInput{
		Email: "owner@tallerflow.pe", Name: "Owner Demo",
		WorkshopName: "Taller Demo", Password: "Temporary secure passphrase 9!",
	}
	first, err := service.CreateOwner(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	firstHash := store.state.credentials[first.UserID].passwordHash
	input.Password = "A different secure passphrase 10!"
	_, err = service.CreateOwner(context.Background(), input)
	if !errors.Is(err, ErrBootstrapAlreadyExists) {
		t.Fatalf("second bootstrap error = %v, want ErrBootstrapAlreadyExists", err)
	}
	if len(store.state.users) != 1 || len(store.state.workshops) != 1 || len(store.state.memberships) != 1 || len(store.state.audit) != 1 || store.state.credentials[first.UserID].passwordHash != firstHash {
		t.Fatal("second bootstrap changed existing owner state")
	}
}

func TestBootstrapRejectsSecondOwnerWithDifferentEmail(t *testing.T) {
	store := newBootstrapStoreFake()
	service := NewBootstrapService(store, NewPasswordHasher(DefaultPasswordParams()))
	first := BootstrapInput{
		Email: "first-owner@tallerflow.pe", Name: "First Owner",
		WorkshopName: "First Workshop", Password: "Temporary secure passphrase 9!",
	}
	if _, err := service.CreateOwner(context.Background(), first); err != nil {
		t.Fatalf("first bootstrap failed: %v", err)
	}

	_, err := service.CreateOwner(context.Background(), BootstrapInput{
		Email: "second-owner@tallerflow.pe", Name: "Second Owner",
		WorkshopName: "Second Workshop", Password: "Another secure passphrase 10!",
	})
	if !errors.Is(err, ErrBootstrapAlreadyExists) {
		t.Fatalf("second bootstrap error = %v, want ErrBootstrapAlreadyExists", err)
	}
	if len(store.state.users) != 1 || len(store.state.workshops) != 1 || len(store.state.memberships) != 1 || len(store.state.audit) != 1 {
		t.Fatal("second bootstrap changed the initial owner state")
	}
}

func TestWebBootstrapCreatesPermanentCredentialThenAppSession(t *testing.T) {
	store := newBootstrapStoreFake()
	service := NewBootstrapServiceWithClock(store, NewPasswordHasher(DefaultPasswordParams()), fixedClock(testNow))
	randomBytes := bytes.Repeat([]byte{7}, 32)
	sessionRepository := &fakeSessionRepository{}
	sessions := NewSessionService(sessionRepository, testPepper, fixedClock(testNow), bytes.NewReader(randomBytes))
	input := BootstrapInput{
		Email: " Web.Owner@TallerFlow.pe ", Name: " Web Owner ",
		WorkshopName: " Taller Web ", Password: "Permanent secure passphrase 9!",
	}

	result, err := service.CreateWebOwner(context.Background(), input, SessionMetadata{IPPrefix: "203.0.113.0/24", UserAgent: "browser"}, sessions)

	if err != nil {
		t.Fatal(err)
	}
	if result.MustChangePassword || result.Token == "" || result.CSRFToken == "" || !result.ExpiresAt.Equal(testNow.Add(8*time.Hour)) {
		t.Fatalf("unexpected web bootstrap result: %+v", result)
	}
	credential := store.state.credentials[result.UserID]
	if credential.mustChange || credential.passwordChangedAt == nil || !credential.passwordChangedAt.Equal(testNow) {
		t.Fatalf("web credential was not marked final at creation: %+v", credential)
	}
	if sessionRepository.inserted == nil || sessionRepository.inserted.UserID != result.UserID {
		t.Fatalf("web bootstrap did not persist exactly one session: %+v", sessionRepository.inserted)
	}
	if sessionRepository.insertForCredentialHash != credential.passwordHash {
		t.Fatal("web bootstrap did not bind the session insert to the committed credential")
	}
	if sessionRepository.inserted.Metadata.IPPrefix != "203.0.113.0/24" {
		t.Fatal("web bootstrap lost session metadata")
	}
	if result.CSRFToken != security.DeriveCSRFToken(testPepper, result.Token) {
		t.Fatal("web bootstrap returned an unbound CSRF token")
	}
}

func TestWebBootstrapDoesNotCreateSessionAfterConcurrentCredentialChange(t *testing.T) {
	store := newBootstrapStoreFake()
	service := NewBootstrapServiceWithClock(store, NewPasswordHasher(DefaultPasswordParams()), fixedClock(testNow))
	sessionRepository := &fakeSessionRepository{insertForCredentialErr: ErrCredentialChanged}
	sessions := NewSessionService(sessionRepository, testPepper, fixedClock(testNow), bytes.NewReader(bytes.Repeat([]byte{9}, 32)))

	_, err := service.CreateWebOwner(context.Background(), BootstrapInput{
		Email: "owner@tallerflow.pe", Name: "Owner Demo", WorkshopName: "Taller Demo",
		Password: "Permanent secure passphrase 9!",
	}, SessionMetadata{}, sessions)

	if !errors.Is(err, ErrCredentialChanged) {
		t.Fatalf("CreateWebOwner() error = %v, want ErrCredentialChanged", err)
	}
	if sessionRepository.inserted != nil {
		t.Fatal("web bootstrap created a session for a stale credential")
	}
}

func TestWebBootstrapLeavesValidClaimedOwnerWhenAppSessionInsertFails(t *testing.T) {
	store := newBootstrapStoreFake()
	service := NewBootstrapServiceWithClock(store, NewPasswordHasher(DefaultPasswordParams()), fixedClock(testNow))
	sessions := NewSessionService(&fakeSessionRepository{insertForCredentialErr: errors.New("session insert failed")}, testPepper, fixedClock(testNow), bytes.NewReader(bytes.Repeat([]byte{8}, 32)))

	_, err := service.CreateWebOwner(context.Background(), BootstrapInput{
		Email: "owner@tallerflow.pe", Name: "Owner Demo", WorkshopName: "Taller Demo",
		Password: "Permanent secure passphrase 9!",
	}, SessionMetadata{}, sessions)

	if err == nil {
		t.Fatal("CreateWebOwner() succeeded despite a session insertion failure")
	}
	if !store.state.bootstrapped || len(store.state.users) != 1 || len(store.state.credentials) != 1 || len(store.state.workshops) != 1 || len(store.state.memberships) != 1 || len(store.state.audit) != 1 {
		t.Fatalf("session failure corrupted the committed owner identity: %+v", store.state)
	}
}

type bootstrapCredential struct {
	passwordHash      string
	mustChange        bool
	passwordChangedAt *time.Time
}
type bootstrapMembership struct {
	userID, workshopID uuid.UUID
	role               string
}
type bootstrapState struct {
	bootstrapped bool
	users        map[string]uuid.UUID
	credentials  map[uuid.UUID]bootstrapCredential
	workshops    map[uuid.UUID]string
	memberships  []bootstrapMembership
	audit        []string
}
type bootstrapStoreFake struct {
	state          bootstrapState
	failMembership bool
}

func newBootstrapStoreFake() *bootstrapStoreFake {
	return &bootstrapStoreFake{state: bootstrapState{
		users: map[string]uuid.UUID{}, credentials: map[uuid.UUID]bootstrapCredential{}, workshops: map[uuid.UUID]string{},
	}}
}

func (s *bootstrapStoreFake) WithinTransaction(ctx context.Context, fn func(BootstrapTx) error) error {
	staged := bootstrapState{
		bootstrapped: s.state.bootstrapped,
		users:        map[string]uuid.UUID{}, credentials: map[uuid.UUID]bootstrapCredential{}, workshops: map[uuid.UUID]string{},
		memberships: append([]bootstrapMembership(nil), s.state.memberships...), audit: append([]string(nil), s.state.audit...),
	}
	for key, value := range s.state.users {
		staged.users[key] = value
	}
	for key, value := range s.state.credentials {
		staged.credentials[key] = value
	}
	for key, value := range s.state.workshops {
		staged.workshops[key] = value
	}
	if err := fn(&bootstrapTxFake{state: &staged, failMembership: s.failMembership}); err != nil {
		return err
	}
	s.state = staged
	return nil
}

type bootstrapTxFake struct {
	state          *bootstrapState
	failMembership bool
}

func (tx *bootstrapTxFake) InsertUser(_ context.Context, email, _ string) (uuid.UUID, error) {
	if _, exists := tx.state.users[email]; exists {
		return uuid.Nil, ErrBootstrapAlreadyExists
	}
	id := uuid.New()
	tx.state.users[email] = id
	return id, nil
}
func (tx *bootstrapTxFake) ClaimInitialOwner(_ context.Context, _ uuid.UUID) error {
	if tx.state.bootstrapped {
		return ErrBootstrapAlreadyExists
	}
	tx.state.bootstrapped = true
	return nil
}
func (tx *bootstrapTxFake) InsertCredential(_ context.Context, userID uuid.UUID, passwordHash string, mustChange bool, passwordChangedAt *time.Time) error {
	tx.state.credentials[userID] = bootstrapCredential{passwordHash, mustChange, passwordChangedAt}
	return nil
}
func (tx *bootstrapTxFake) InsertWorkshop(_ context.Context, name, timezone string) (uuid.UUID, error) {
	id := uuid.New()
	tx.state.workshops[id] = name + "|" + timezone
	return id, nil
}
func (tx *bootstrapTxFake) InsertMembership(_ context.Context, userID, workshopID uuid.UUID, role string) error {
	if tx.failMembership {
		return errors.New("membership insert failed")
	}
	tx.state.memberships = append(tx.state.memberships, bootstrapMembership{userID, workshopID, role})
	return nil
}
func (tx *bootstrapTxFake) InsertAudit(_ context.Context, event string, _, _ uuid.UUID) error {
	tx.state.audit = append(tx.state.audit, event)
	return nil
}
