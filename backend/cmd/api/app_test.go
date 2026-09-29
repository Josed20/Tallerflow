package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Josed20/Tallerflow/backend/platform/config"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBuildApplicationRegistersAuthWorkshopAndPasswordResetRoutes(t *testing.T) {
	db, _, cleanup := applicationDatabase(t)
	defer cleanup()
	app, err := buildApplication(applicationConfig(), withDatabaseOpener(func(string) (*gorm.DB, error) { return db, nil }))
	require.NoError(t, err)

	login := httptest.NewRecorder()
	app.handler.ServeHTTP(login, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	require.NotEqual(t, http.StatusNotFound, login.Code)
	me := httptest.NewRecorder()
	app.handler.ServeHTTP(me, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	require.Equal(t, http.StatusUnauthorized, me.Code)
	recovery := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-resets", nil)
	app.handler.ServeHTTP(recovery, request)
	require.Equal(t, http.StatusBadRequest, recovery.Code)
}

func TestBuildApplicationReadinessUsesDatabase(t *testing.T) {
	db, mock, cleanup := applicationDatabase(t)
	defer cleanup()
	mock.ExpectPing()
	app, err := buildApplication(applicationConfig(), withDatabaseOpener(func(string) (*gorm.DB, error) { return db, nil }))
	require.NoError(t, err)
	response := httptest.NewRecorder()

	app.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health/ready", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestBuildApplicationRejectsInvalidTrustedProxy(t *testing.T) {
	db, _, cleanup := applicationDatabase(t)
	defer cleanup()
	cfg := applicationConfig()
	cfg.TrustedProxies = []string{"not-a-network"}

	_, err := buildApplication(cfg, withDatabaseOpener(func(string) (*gorm.DB, error) { return db, nil }))

	require.ErrorContains(t, err, "TF_TRUSTED_PROXIES")
}

func TestBuildApplicationRejectsUnsafeSMTPConfiguration(t *testing.T) {
	db, _, cleanup := applicationDatabase(t)
	defer cleanup()
	cfg := applicationConfig()
	cfg.SMTPHost = "smtp.example.com"
	cfg.SMTPPort = "587"
	cfg.SMTPUsername = "smtp-user"
	cfg.SMTPPassword = "smtp-password"
	cfg.SMTPRequireTLS = false

	_, err := buildApplication(cfg, withDatabaseOpener(func(string) (*gorm.DB, error) { return db, nil }))

	require.ErrorContains(t, err, "password reset delivery")
}

func TestBuildApplicationClosesDatabase(t *testing.T) {
	db, mock, _ := applicationDatabase(t)
	mock.ExpectClose()
	app, err := buildApplication(applicationConfig(), withDatabaseOpener(func(string) (*gorm.DB, error) { return db, nil }))
	require.NoError(t, err)

	require.NoError(t, app.close())
	require.NoError(t, mock.ExpectationsWereMet())
}

func applicationConfig() config.Config {
	return config.Config{
		Environment: "development", HTTPAddress: ":0", DatabaseURL: "postgres://ignored",
		SessionPepper: "test-session-pepper", AllowedOrigin: "http://localhost:8080",
		PasswordResetBaseURL: "http://localhost:8080", SMTPHost: "localhost", SMTPPort: "1025",
		SMTPFromAddress: "soporte@tallerflow.pe",
	}
}

func applicationDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	return db, mock, func() { _ = sqlDB.Close() }
}
