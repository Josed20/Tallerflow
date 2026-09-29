package auth

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestPostgresIntegrationAuthenticationAndTenantIsolation(t *testing.T) {
	databaseURL := os.Getenv("TF_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TF_TEST_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	db, err := platformdb.Open(databaseURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(db)) })
	require.NoError(t, platformdb.Ping(ctx, db))

	now := time.Now().UTC().Truncate(time.Microsecond)
	repository, err := NewPostgresRepository(db, func() time.Time { return now })
	require.NoError(t, err)
	if ownerEmail, ownerPassword := os.Getenv("E2E_OWNER_EMAIL"), os.Getenv("E2E_OWNER_PASSWORD"); ownerEmail != "" && ownerPassword != "" {
		t.Run("bootstrap owner can complete the real login path", func(t *testing.T) {
			credential, err := repository.FindByEmail(ctx, ownerEmail)
			require.NoError(t, err)
			matches, err := NewPasswordHasher(DefaultPasswordParams()).Verify(credential.PasswordHash, ownerPassword)
			require.NoError(t, err)
			require.True(t, matches, "bootstrap stored a credential that does not match the supplied password")
			membership, err := repository.ResolveActive(ctx, credential.UserID)
			require.NoError(t, err)
			require.Equal(t, "OWNER", membership.Role)

			sessions := NewSessionService(repository, []byte("integration-session-pepper"), func() time.Time { return now }, nil)
			service := NewAuthService(repository, repository, sessions, NewPasswordHasher(DefaultPasswordParams()), repository)
			loggedIn, err := service.Login(ctx, ownerEmail, ownerPassword, "203.0.113.200")
			require.NoError(t, err)
			require.True(t, loggedIn.MustChangePassword)
			require.Equal(t, membership.WorkshopID, loggedIn.Principal.WorkshopID)
		})
	}
	tenantRunner, err := platformdb.NewTenantRunner(db)
	require.NoError(t, err)
	userID, workshopID, otherWorkshopID := uuid.New(), uuid.New(), uuid.New()
	email := "integration-" + userID.String() + "@example.com"
	oldHash, replacementHash := "integration-old-hash", "integration-new-hash"
	require.NoError(t, db.WithContext(ctx).Exec(
		`INSERT INTO users (id, email, name, status) VALUES (?, ?, ?, 'ACTIVE')`, userID, email, "Integration Owner",
	).Error)
	require.NoError(t, db.WithContext(ctx).Exec(
		`INSERT INTO user_credentials (user_id, password_hash, must_change_password) VALUES (?, ?, true)`, userID, oldHash,
	).Error)
	require.NoError(t, seedTenant(ctx, tenantRunner, workshopID, userID, true))
	require.NoError(t, seedTenant(ctx, tenantRunner, otherWorkshopID, userID, false))

	t.Run("credential session and membership persistence", func(t *testing.T) {
		credential, err := repository.FindByEmail(ctx, "  "+email+"  ")
		require.NoError(t, err)
		require.Equal(t, userID, credential.UserID)
		require.Equal(t, "Integration Owner", credential.DisplayName)
		membership, err := repository.ResolveActive(ctx, userID)
		require.NoError(t, err)
		require.Equal(t, workshopID, membership.WorkshopID)

		created := NewSession{UserID: userID, TokenHash: [32]byte{1, 2, 3}, CSRFTokenHash: [32]byte{4, 5, 6}, CreatedAt: now, ExpiresAt: now.Add(time.Hour)}
		session, err := repository.Insert(ctx, created)
		require.NoError(t, err)
		found, err := repository.FindActiveByTokenHash(ctx, created.TokenHash, now)
		require.NoError(t, err)
		require.Equal(t, session.ID, found.ID)
	})

	t.Run("stale verified password cannot create a session", func(t *testing.T) {
		require.NoError(t, db.WithContext(ctx).Exec(`UPDATE user_credentials SET password_hash = ? WHERE user_id = ?`, replacementHash, userID).Error)
		created := NewSession{UserID: userID, TokenHash: [32]byte{7}, CSRFTokenHash: [32]byte{8}, CreatedAt: now.Add(time.Second), ExpiresAt: now.Add(time.Hour)}

		_, err := repository.InsertForCredential(ctx, oldHash, created)

		require.ErrorIs(t, err, ErrCredentialChanged)
		var count int64
		require.NoError(t, db.Raw(`SELECT count(*) FROM user_sessions WHERE token_hash = ?`, created.TokenHash[:]).Scan(&count).Error)
		require.Zero(t, count)
	})

	t.Run("rotating emails cannot bypass the IP bucket", func(t *testing.T) {
		ip := "198.51.100." + time.Now().Format("150405.000000")
		for index := 0; index < 5; index++ {
			require.NoError(t, repository.RecordFailure(ctx, uuid.NewString()+"@example.com", ip))
		}
		allowed, err := repository.Allow(ctx, uuid.NewString()+"@example.com", ip)
		require.NoError(t, err)
		require.False(t, allowed)
	})

	t.Run("tenant context cannot read another workshop", func(t *testing.T) {
		require.NoError(t, tenantRunner.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
			var count int64
			if err := tx.Raw(`SELECT count(*) FROM workshops WHERE id = ?`, otherWorkshopID).Scan(&count).Error; err != nil {
				return err
			}
			if count != 0 {
				return errors.New("cross-tenant workshop became visible")
			}
			return nil
		}))
	})
}

func seedTenant(ctx context.Context, runner platformdb.TenantRunner, workshopID, userID uuid.UUID, withMembership bool) error {
	return runner.WithinTenant(ctx, workshopID, func(tx *gorm.DB) error {
		if err := tx.Exec(`INSERT INTO workshops (id, name, timezone) VALUES (?, ?, 'America/Lima')`, workshopID, "Integration Workshop").Error; err != nil {
			return err
		}
		if withMembership {
			return tx.Exec(`INSERT INTO workshop_members (workshop_id, user_id, role, status) VALUES (?, ?, 'OWNER', 'ACTIVE')`, workshopID, userID).Error
		}
		return nil
	})
}
