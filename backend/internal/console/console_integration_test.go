package console

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Josed20/Tallerflow/backend/internal/auth"
	platformdb "github.com/Josed20/Tallerflow/backend/platform/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

const integrationTOTPKey = "console integration key that is long enough"

func integrationDatabases(t *testing.T) (console *gorm.DB, admin *gorm.DB) {
	t.Helper()
	consoleURL, adminURL := os.Getenv("TF_TEST_CONSOLE_DATABASE_URL"), os.Getenv("TF_TEST_ADMIN_DATABASE_URL")
	if consoleURL == "" || adminURL == "" {
		t.Skip("TF_TEST_CONSOLE_DATABASE_URL and TF_TEST_ADMIN_DATABASE_URL are not configured")
	}
	console, err := platformdb.Open(consoleURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(console)) })
	admin, err = platformdb.Open(adminURL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, platformdb.Close(admin)) })
	return console, admin
}

// The console runs as tallerflow_console, so this goes through the same role,
// grants and SQL functions as production: login, CSRF and a status change.
func TestPostgresIntegrationConsoleSuspendsAndReactivatesAWorkshop(t *testing.T) {
	consoleDB, adminDB := integrationDatabases(t)
	ctx := context.Background()
	service, err := NewService(consoleDB, integrationTOTPKey)
	require.NoError(t, err)

	email := "console-" + uuid.NewString() + "@example.com"
	password := "Console integration password 2026!"
	secret, err := GenerateSecret()
	require.NoError(t, err)
	hash, err := auth.NewPasswordHasher(auth.DefaultPasswordParams()).Hash(password)
	require.NoError(t, err)
	ciphertext, err := EncryptSecret(integrationTOTPKey, secret)
	require.NoError(t, err)
	require.NoError(t, consoleDB.Exec("INSERT INTO console_admins (id, email, password_hash, totp_secret_ciphertext) VALUES (?, ?, ?, ?)",
		uuid.New(), email, hash, ciphertext).Error)

	token, csrf, _, err := service.Login(ctx, email, password, totpAt(secret, time.Now()))
	require.NoError(t, err)
	session, err := service.Authenticate(ctx, token)
	require.NoError(t, err)
	require.True(t, service.ValidateCSRF(ctx, session.ID, csrf), "the token issued at login must pass")
	require.False(t, service.ValidateCSRF(ctx, session.ID, csrf+"x"))
	require.False(t, service.ValidateCSRF(ctx, uuid.New(), csrf), "another session's token must not pass")

	workshopID := uuid.New()
	require.NoError(t, adminDB.Exec("INSERT INTO workshops (id, name, timezone) VALUES (?, ?, 'America/Lima')", workshopID, "Taller Consola "+workshopID.String()[:8]).Error)

	_, err = service.SetStatus(ctx, session.AdminID, workshopID, "SUSPENDED", "Prueba de integración", "203.0.113.7")
	require.NoError(t, err)
	require.Equal(t, "SUSPENDED", workshopStatus(t, adminDB, workshopID))
	_, err = service.SetStatus(ctx, session.AdminID, workshopID, "ACTIVE", "Prueba terminada", "203.0.113.7")
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", workshopStatus(t, adminDB, workshopID))

	events, err := service.Audit(ctx, 50)
	require.NoError(t, err)
	recorded := 0
	for _, event := range events {
		if event.WorkshopID != nil && *event.WorkshopID == workshopID {
			recorded++
		}
	}
	require.Equal(t, 2, recorded, "both status changes are audited")
}

func workshopStatus(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var status string
	require.NoError(t, db.Raw("SELECT status FROM workshops WHERE id = ?", id).Row().Scan(&status))
	return status
}
