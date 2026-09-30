package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Josed20/Tallerflow/backend/internal/auth"
	"github.com/Josed20/Tallerflow/backend/internal/onboarding"
	"github.com/Josed20/Tallerflow/backend/platform/config"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBuildApplicationRegistersAuthWorkshopAndPasswordResetRoutes(t *testing.T) {
	db, _, cleanup := applicationDatabase(t)
	defer cleanup()
	app, err := buildApplication(applicationConfig(), applicationTestOptions(db)...)
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

func TestBuildApplicationRegistersProtectedTeamRoutes(t *testing.T) {
	db, _, cleanup := applicationDatabase(t)
	defer cleanup()
	app, err := buildApplication(applicationConfig(), applicationTestOptions(db)...)
	require.NoError(t, err)

	team := httptest.NewRecorder()
	app.handler.ServeHTTP(team, httptest.NewRequest(http.MethodGet, "/api/v1/team", nil))
	require.Equal(t, http.StatusUnauthorized, team.Code)
}

func TestBuildApplicationRegistersOnboardingRoutesWithSeparateBootstrapURL(t *testing.T) {
	db, _, cleanup := applicationDatabase(t)
	defer cleanup()
	var openedURL string
	options := applicationTestOptions(db)
	options = append(options, withOnboardingModuleBuilder(func(databaseURL string, _ *auth.SessionService) (onboardingModule, error) {
		openedURL = databaseURL
		return testOnboardingModule(), nil
	}))

	app, err := buildApplication(applicationConfig(), options...)
	require.NoError(t, err)
	response := httptest.NewRecorder()
	app.handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/onboarding/status", nil))

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "postgres://bootstrap-ignored", openedURL)
	require.NotEqual(t, applicationConfig().DatabaseURL, openedURL)
}

func TestBuildApplicationReadinessUsesDatabase(t *testing.T) {
	db, mock, cleanup := applicationDatabase(t)
	defer cleanup()
	mock.ExpectPing()
	app, err := buildApplication(applicationConfig(), applicationTestOptions(db)...)
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

	_, err := buildApplication(cfg, applicationTestOptions(db)...)

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
	bootstrapClosed := false
	options := applicationTestOptions(db)
	options = append(options, withOnboardingModuleBuilder(func(string, *auth.SessionService) (onboardingModule, error) {
		module := testOnboardingModule()
		module.close = func() error { bootstrapClosed = true; return nil }
		return module, nil
	}))
	app, err := buildApplication(applicationConfig(), options...)
	require.NoError(t, err)

	require.NoError(t, app.close())
	require.NoError(t, mock.ExpectationsWereMet())
	require.True(t, bootstrapClosed)
}

func TestBuildPostgresOnboardingModuleDoesNotExposeInvalidDatabaseURL(t *testing.T) {
	const secret = "bootstrap-secret-fixture"
	_, err := buildPostgresOnboardingModule("postgres://bootstrap:"+secret+"@%zz/tallerflow", nil)

	require.Error(t, err)
	require.ErrorContains(t, err, "TF_BOOTSTRAP_DATABASE_URL")
	require.False(t, strings.Contains(err.Error(), secret), "configuration error leaked the bootstrap password: %v", err)
}

func applicationConfig() config.Config {
	return config.Config{
		Environment: "development", HTTPAddress: ":0", DatabaseURL: "postgres://runtime-ignored", BootstrapDatabaseURL: "postgres://bootstrap-ignored",
		SessionPepper: "test-session-pepper", AllowedOrigin: "http://localhost:8080",
		PasswordResetBaseURL: "http://localhost:8080", SMTPHost: "localhost", SMTPPort: "1025",
		SMTPFromAddress: "soporte@tallerflow.pe",
	}
}

func applicationTestOptions(db *gorm.DB) []applicationOption {
	return []applicationOption{
		withDatabaseOpener(func(string) (*gorm.DB, error) { return db, nil }),
		withOnboardingModuleBuilder(func(string, *auth.SessionService) (onboardingModule, error) { return testOnboardingModule(), nil }),
	}
}

func testOnboardingModule() onboardingModule {
	return onboardingModule{
		service: appOnboardingFake{},
		ping:    func(context.Context) error { return nil },
		close:   func() error { return nil },
	}
}

type appOnboardingFake struct{}

func (appOnboardingFake) Status(context.Context) (onboarding.Status, error) {
	return onboarding.Status{Available: true}, nil
}
func (appOnboardingFake) Create(context.Context, onboarding.Input, auth.SessionMetadata) (onboarding.Result, error) {
	return onboarding.Result{}, nil
}

func applicationDatabase(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, func()) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	return db, mock, func() { _ = sqlDB.Close() }
}
