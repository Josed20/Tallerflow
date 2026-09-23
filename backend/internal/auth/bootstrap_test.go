package auth

import (
	"context"
	"errors"
	"testing"

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

type bootstrapCredential struct {
	passwordHash string
	mustChange   bool
}
type bootstrapMembership struct {
	userID, workshopID uuid.UUID
	role               string
}
type bootstrapState struct {
	users       map[string]uuid.UUID
	credentials map[uuid.UUID]bootstrapCredential
	workshops   map[uuid.UUID]string
	memberships []bootstrapMembership
	audit       []string
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
		users: map[string]uuid.UUID{}, credentials: map[uuid.UUID]bootstrapCredential{}, workshops: map[uuid.UUID]string{},
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
func (tx *bootstrapTxFake) InsertCredential(_ context.Context, userID uuid.UUID, passwordHash string, mustChange bool) error {
	tx.state.credentials[userID] = bootstrapCredential{passwordHash, mustChange}
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
