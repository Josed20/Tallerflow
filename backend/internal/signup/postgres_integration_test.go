package signup

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func integrationRepository(t *testing.T) (*PostgresRepository, *gorm.DB, *platformdb.GormTenantRunner) {
	t.Helper()
	databaseURL := os.Getenv("TF_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TF_TEST_DATABASE_URL is not configured")
	}
	db, err := platformdb.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })
	tenants, err := platformdb.NewTenantRunner(db)
	require.NoError(t, err)
	repository, err := NewPostgresRepository(db, tenants)
	require.NoError(t, err)
	return repository, db, tenants
}

func pendingAccount(t *testing.T, email, workshop, passwordHash string) (PendingAccount, string) {
	t.Helper()
	raw, digest, err := GenerateToken()
	require.NoError(t, err)
	return PendingAccount{
		Email: email, OwnerName: "Owner " + workshop, WorkshopName: workshop, PasswordHash: passwordHash,
		TokenHash: digest, ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}, raw
}

func digestOf(t *testing.T, raw string) []byte {
	t.Helper()
	digest, ok := HashToken(raw)
	require.True(t, ok)
	return digest
}

func TestPostgresIntegrationSignupVerifiesOnceAndUnlocksLogin(t *testing.T) {
	repository, db, _ := integrationRepository(t)
	ctx := context.Background()
	email := "signup-" + uuid.NewString() + "@example.com"
	account, raw := pendingAccount(t, email, "Taller Integración", "argon-hash-1")

	issued, err := repository.Register(ctx, account)
	require.NoError(t, err)
	require.True(t, issued)

	credentials, err := auth.NewPostgresRepository(db, time.Now)
	require.NoError(t, err)
	credential, err := credentials.FindByEmail(ctx, email)
	require.NoError(t, err)
	require.True(t, credential.PendingEmailVerification, "a new signup cannot sign in before verifying")
	require.False(t, credential.MustChangePassword)

	pendingHash, err := repository.PendingPasswordHash(ctx, digestOf(t, raw))
	require.NoError(t, err)
	require.Equal(t, "argon-hash-1", pendingHash, "reading the password hash does not use the link")

	userID, err := repository.ConsumeToken(ctx, digestOf(t, raw))
	require.NoError(t, err)
	require.Equal(t, credential.UserID, userID)

	_, err = repository.ConsumeToken(ctx, digestOf(t, raw))
	require.ErrorIs(t, err, ErrTokenInvalid, "a used link cannot verify again")
	_, err = repository.PendingPasswordHash(ctx, digestOf(t, raw))
	require.ErrorIs(t, err, ErrTokenInvalid)

	credential, err = credentials.FindByEmail(ctx, email)
	require.NoError(t, err)
	require.False(t, credential.PendingEmailVerification)
	membership, err := credentials.ResolveActive(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, "OWNER", membership.Role)

	var terms struct{ Accepted bool }
	require.NoError(t, db.Raw(`SELECT terms_accepted_at IS NOT NULL AS accepted FROM users WHERE id = ?`, userID).Scan(&terms).Error)
	require.True(t, terms.Accepted)
}

func TestPostgresIntegrationTwoSignedUpWorkshopsCannotSeeEachOther(t *testing.T) {
	repository, db, tenants := integrationRepository(t)
	ctx := context.Background()
	first, _ := pendingAccount(t, "first-"+uuid.NewString()+"@example.com", "Taller Uno", "hash-uno")
	second, _ := pendingAccount(t, "second-"+uuid.NewString()+"@example.com", "Taller Dos", "hash-dos")
	for _, account := range []PendingAccount{first, second} {
		issued, err := repository.Register(ctx, account)
		require.NoError(t, err)
		require.True(t, issued)
	}
	firstWorkshop := workshopForEmail(t, repository, db, first.Email)
	secondWorkshop := workshopForEmail(t, repository, db, second.Email)
	require.NotEqual(t, firstWorkshop, secondWorkshop)

	require.NoError(t, tenants.WithinTenant(ctx, firstWorkshop, func(tx *gorm.DB) error {
		var visible []struct{ ID uuid.UUID }
		require.NoError(t, tx.Raw(`SELECT id FROM workshops`).Scan(&visible).Error)
		require.Len(t, visible, 1)
		require.Equal(t, firstWorkshop, visible[0].ID)
		var members int64
		require.NoError(t, tx.Raw(`SELECT count(*) FROM workshop_members WHERE workshop_id = ?`, secondWorkshop).Scan(&members).Error)
		require.Zero(t, members, "the other workshop's owner is invisible")
		return nil
	}))
}

func TestPostgresIntegrationVerifiedAddressIsNeverOverwritten(t *testing.T) {
	repository, db, _ := integrationRepository(t)
	ctx := context.Background()
	email := "verified-" + uuid.NewString() + "@example.com"
	account, raw := pendingAccount(t, email, "Taller Original", "original-hash")
	_, err := repository.Register(ctx, account)
	require.NoError(t, err)
	_, err = repository.ConsumeToken(ctx, digestOf(t, raw))
	require.NoError(t, err)

	again, _ := pendingAccount(t, email, "Taller Intruso", "intruder-hash")
	issued, err := repository.Register(ctx, again)
	require.NoError(t, err)
	require.False(t, issued)
	require.Equal(t, "original-hash", passwordHashOf(t, db, email))

	_, issued, err = repository.ReissueToken(ctx, email, digestOf(t, mustToken(t)), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.False(t, issued, "a verified account gets no new link")
}

func TestPostgresIntegrationASecondSignupReplacesAnUnverifiedAccount(t *testing.T) {
	repository, db, tenants := integrationRepository(t)
	ctx := context.Background()
	email := "pending-" + uuid.NewString() + "@example.com"
	squatter, squatterToken := pendingAccount(t, email, "Taller Ajeno", "squatter-hash")
	_, err := repository.Register(ctx, squatter)
	require.NoError(t, err)

	owner, ownerToken := pendingAccount(t, email, "Taller Real", "owner-hash")
	issued, err := repository.Register(ctx, owner)
	require.NoError(t, err)
	require.True(t, issued)
	require.Equal(t, "owner-hash", passwordHashOf(t, db, email), "the last registration's password wins")

	_, err = repository.ConsumeToken(ctx, digestOf(t, squatterToken))
	require.ErrorIs(t, err, ErrTokenInvalid, "the earlier link stops working")
	_, err = repository.ConsumeToken(ctx, digestOf(t, ownerToken))
	require.NoError(t, err)

	workshopID := workshopForEmail(t, repository, db, email)
	require.NoError(t, tenants.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		var name string
		require.NoError(t, tx.Raw(`SELECT name FROM workshops WHERE id = ?`, workshopID).Scan(&name).Error)
		require.Equal(t, "Taller Real", name)
		return nil
	}))
}

func TestPostgresIntegrationExpiredTamperedAndReissuedTokensAreRejected(t *testing.T) {
	repository, db, _ := integrationRepository(t)
	ctx := context.Background()
	email := "expiry-" + uuid.NewString() + "@example.com"
	account, raw := pendingAccount(t, email, "Taller Vencido", "hash")
	_, err := repository.Register(ctx, account)
	require.NoError(t, err)

	_, err = repository.ConsumeToken(ctx, digestOf(t, mustToken(t)))
	require.ErrorIs(t, err, ErrTokenInvalid, "an unknown or tampered token is rejected")

	require.NoError(t, db.Exec(`UPDATE email_verification_tokens SET created_at = now() - interval '2 days', expires_at = now() - interval '1 day' WHERE token_hash = ?`, digestOf(t, raw)).Error)
	_, err = repository.ConsumeToken(ctx, digestOf(t, raw))
	require.ErrorIs(t, err, ErrTokenInvalid, "an expired token is rejected")

	fresh := mustToken(t)
	recipient, issued, err := repository.ReissueToken(ctx, email, digestOf(t, fresh), time.Now().UTC().Add(24*time.Hour))
	require.NoError(t, err)
	require.True(t, issued)
	require.Equal(t, "Owner Taller Vencido", recipient.Name)
	_, err = repository.ConsumeToken(ctx, digestOf(t, fresh))
	require.NoError(t, err)

	_, issued, err = repository.ReissueToken(ctx, "nobody-"+uuid.NewString()+"@example.com", digestOf(t, mustToken(t)), time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.False(t, issued)
}

func TestPostgresIntegrationVerificationIsSingleUseUnderConcurrency(t *testing.T) {
	repository, _, _ := integrationRepository(t)
	ctx := context.Background()
	account, raw := pendingAccount(t, "race-"+uuid.NewString()+"@example.com", "Taller Carrera", "hash")
	_, err := repository.Register(ctx, account)
	require.NoError(t, err)

	var succeeded atomic.Int32
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			if _, err := repository.ConsumeToken(ctx, digestOf(t, raw)); err == nil {
				succeeded.Add(1)
			}
		}()
	}
	group.Wait()
	require.Equal(t, int32(1), succeeded.Load())
}

func mustToken(t *testing.T) string {
	t.Helper()
	raw, _, err := GenerateToken()
	require.NoError(t, err)
	return raw
}

func passwordHashOf(t *testing.T, db *gorm.DB, email string) string {
	t.Helper()
	var hash string
	require.NoError(t, db.Raw(`SELECT c.password_hash FROM user_credentials AS c JOIN users AS u ON u.id = c.user_id WHERE u.email = ?`, email).Scan(&hash).Error)
	return hash
}

func workshopForEmail(t *testing.T, repository *PostgresRepository, db *gorm.DB, email string) uuid.UUID {
	t.Helper()
	var user struct{ ID uuid.UUID }
	require.NoError(t, db.Raw(`SELECT id FROM users WHERE email = ?`, email).Scan(&user).Error)
	workshopID, ok, err := repository.workshopOf(context.Background(), user.ID)
	require.NoError(t, err)
	require.True(t, ok)
	return workshopID
}
