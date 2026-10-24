package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadRejectsMissingSecretsWithoutLeakingConfiguredValues(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		pepper      string
		missing     string
		secret      string
	}{
		{
			name:        "database URL",
			databaseURL: "",
			pepper:      "session-pepper-that-must-not-leak",
			missing:     "TF_DATABASE_URL",
			secret:      "session-pepper-that-must-not-leak",
		},
		{
			name:        "session pepper",
			databaseURL: "postgres://tallerflow:database-password-that-must-not-leak@localhost:5432/tallerflow",
			pepper:      "",
			missing:     "TF_SESSION_PEPPER",
			secret:      "database-password-that-must-not-leak",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TF_DATABASE_URL", tt.databaseURL)
			t.Setenv("TF_SESSION_PEPPER", tt.pepper)

			_, err := Load()

			require.ErrorContains(t, err, tt.missing)
			require.NotContains(t, err.Error(), tt.secret)
		})
	}
}

func TestLoadUsesDefaultHTTPAddress(t *testing.T) {
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow:password@localhost:5432/tallerflow")
	t.Setenv("TF_BOOTSTRAP_DATABASE_URL", "postgres://tallerflow_bootstrap:password@localhost:5432/tallerflow")
	t.Setenv("TF_SESSION_PEPPER", "test-session-pepper")
	t.Setenv("TF_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("TF_HTTP_ADDRESS", "")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.HTTPAddress)
}

func TestLoadRequiresBootstrapDatabaseURLWithoutLeakingConfiguredSecrets(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_BOOTSTRAP_DATABASE_URL", "")
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow:runtime-secret@localhost:5432/tallerflow")

	_, err := Load()

	require.ErrorContains(t, err, "TF_BOOTSTRAP_DATABASE_URL")
	require.NotContains(t, err.Error(), "runtime-secret")
}

func TestLoadKeepsBootstrapDatabaseSeparateFromRuntimeDatabase(t *testing.T) {
	setValidEnvironment(t)
	bootstrapURL := "postgres://tallerflow_bootstrap:bootstrap-secret@localhost:5432/tallerflow"
	t.Setenv("TF_BOOTSTRAP_DATABASE_URL", bootstrapURL)

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, bootstrapURL, cfg.BootstrapDatabaseURL)
	require.NotEqual(t, cfg.DatabaseURL, cfg.BootstrapDatabaseURL)
}

func TestLoadRequiresAllowedOrigin(t *testing.T) {
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow:password@localhost:5432/tallerflow")
	t.Setenv("TF_SESSION_PEPPER", "test-session-pepper")
	t.Setenv("TF_ALLOWED_ORIGIN", "")

	_, err := Load()

	require.ErrorContains(t, err, "TF_ALLOWED_ORIGIN")
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_ENVIRONMENT", "prodution")

	_, err := Load()

	require.ErrorContains(t, err, "TF_ENVIRONMENT")
	require.NotContains(t, err.Error(), "prodution")
}

func TestLoadRejectsInsecureProductionOrigin(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_ENVIRONMENT", "production")
	t.Setenv("TF_ALLOWED_ORIGIN", "http://app.tallerflow.test")

	_, err := Load()

	require.ErrorContains(t, err, "TF_ALLOWED_ORIGIN")
	require.NotContains(t, err.Error(), "app.tallerflow.test")
}

func TestLoadParsesTrustedProxies(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_TRUSTED_PROXIES", " 172.16.0.0/12, 10.0.0.8 ")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, []string{"172.16.0.0/12", "10.0.0.8"}, cfg.TrustedProxies)
}

func TestLoadDefaultsTrustedProxyOnlyForProduction(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_TRUSTED_PROXIES", "")

	development, err := Load()
	require.NoError(t, err)
	require.Empty(t, development.TrustedProxies)

	setValidProductionEnvironment(t)
	t.Setenv("TF_TRUSTED_PROXIES", "")
	production, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"172.16.0.0/12"}, production.TrustedProxies)
}

func TestLoadDefaultsPasswordResetDeliveryForDevelopment(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_SMTP_HOST", "")
	t.Setenv("TF_SMTP_PORT", "")
	t.Setenv("TF_SMTP_FROM", "")
	t.Setenv("TF_SMTP_REQUIRE_TLS", "")
	t.Setenv("TF_PASSWORD_RESET_BASE_URL", "")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, "localhost", cfg.SMTPHost)
	require.Equal(t, "1025", cfg.SMTPPort)
	require.Equal(t, "soporte@tallerflow.pe", cfg.SMTPFromAddress)
	require.False(t, cfg.SMTPRequireTLS)
	require.Equal(t, cfg.AllowedOrigin, cfg.PasswordResetBaseURL)
}

func TestLoadRejectsInvalidSMTPTLSFlagWithoutLeakingValue(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_SMTP_REQUIRE_TLS", "not-a-boolean-secret")

	_, err := Load()

	require.ErrorContains(t, err, "TF_SMTP_REQUIRE_TLS")
	require.NotContains(t, err.Error(), "not-a-boolean-secret")
}

func TestLoadRejectsDisabledSMTPTLSInProduction(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_ENVIRONMENT", "production")
	setValidProductionEnvironment(t)
	t.Setenv("TF_SMTP_REQUIRE_TLS", "false")

	_, err := Load()

	require.ErrorContains(t, err, "TF_SMTP_REQUIRE_TLS")
}

func TestLoadRejectsInsecureProductionSecrets(t *testing.T) {
	tests := []struct {
		name     string
		variable string
		value    string
	}{
		{name: "short session pepper", variable: "TF_SESSION_PEPPER", value: "too-short"},
		{name: "runtime role", variable: "TF_DATABASE_URL", value: "postgres://postgres:long-enough-database-secret@db:5432/tallerflow?sslmode=require"},
		{name: "runtime TLS disabled", variable: "TF_DATABASE_URL", value: "postgres://tallerflow_app:long-enough-database-secret@db:5432/tallerflow?sslmode=disable"},
		{name: "bootstrap default password", variable: "TF_BOOTSTRAP_DATABASE_URL", value: "postgres://tallerflow_bootstrap:local-bootstrap-change-me@db:5432/tallerflow?sslmode=require"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidProductionEnvironment(t)
			t.Setenv(tt.variable, tt.value)

			_, err := Load()

			require.Error(t, err)
			require.NotContains(t, err.Error(), tt.value)
		})
	}
}

func TestLoadAcceptsHardenedProductionConfiguration(t *testing.T) {
	setValidProductionEnvironment(t)

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, "production", cfg.Environment)
}

func TestLoadAcceptsPlaintextToComposeDatabaseInProduction(t *testing.T) {
	// Same shape as the VPS .env in docs/runbooks/vps.md.
	setValidProductionEnvironment(t)
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow_app:runtime-database-secret-123@postgres:5432/tallerflow?sslmode=disable")
	t.Setenv("TF_BOOTSTRAP_DATABASE_URL", "postgres://tallerflow_bootstrap:bootstrap-database-secret-123@postgres:5432/tallerflow?sslmode=disable")
	t.Setenv("TF_CONSOLE_DATABASE_URL", "postgres://tallerflow_console:console-database-secret-123@postgres:5432/tallerflow?sslmode=disable")

	_, err := Load()

	require.NoError(t, err)
}

func TestLoadRejectsInvalidTrustedProxyWithoutLeakingValue(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_TRUSTED_PROXIES", "not-a-network-secret")

	_, err := Load()

	require.ErrorContains(t, err, "TF_TRUSTED_PROXIES")
	require.False(t, strings.Contains(err.Error(), "not-a-network-secret"))
}

func TestLoadDefaultsMobileOriginsPerEnvironment(t *testing.T) {
	setValidEnvironment(t)

	development, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"http://localhost", "https://localhost", "capacitor://localhost"}, development.MobileOrigins)

	setValidProductionEnvironment(t)
	production, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"https://localhost", "capacitor://localhost"}, production.MobileOrigins)
}

func TestLoadParsesMobileOrigins(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_MOBILE_ORIGINS", " HTTPS://LocalHost , capacitor://localhost ")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, []string{"https://localhost", "capacitor://localhost"}, cfg.MobileOrigins)
}

func TestLoadRejectsInvalidMobileOriginsWithoutLeakingValue(t *testing.T) {
	for name, value := range map[string]string{
		"wildcard":     "*",
		"path":         "https://localhost/app-secret",
		"empty member": "https://localhost,,capacitor://localhost",
		"other scheme": "ftp://mobile-secret",
	} {
		t.Run(name, func(t *testing.T) {
			setValidEnvironment(t)
			t.Setenv("TF_MOBILE_ORIGINS", value)

			_, err := Load()

			require.ErrorContains(t, err, "TF_MOBILE_ORIGINS")
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestLoadRejectsInsecureMobileOriginInProduction(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("TF_MOBILE_ORIGINS", "http://localhost")

	_, err := Load()

	require.ErrorContains(t, err, "TF_MOBILE_ORIGINS")
}

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("TF_MOBILE_ORIGINS", "")
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow:password@localhost:5432/tallerflow")
	t.Setenv("TF_BOOTSTRAP_DATABASE_URL", "postgres://tallerflow_bootstrap:password@localhost:5432/tallerflow")
	t.Setenv("TF_SESSION_PEPPER", "test-session-pepper")
	t.Setenv("TF_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("TF_ENVIRONMENT", "development")
	t.Setenv("TF_PASSWORD_RESET_BASE_URL", "")
	t.Setenv("TF_SMTP_HOST", "")
	t.Setenv("TF_SMTP_PORT", "")
	t.Setenv("TF_SMTP_USER", "")
	t.Setenv("TF_SMTP_PASSWORD", "")
	t.Setenv("TF_SMTP_FROM", "")
	t.Setenv("TF_SMTP_REQUIRE_TLS", "")
	t.Setenv("TF_R2_ENDPOINT", "https://account.r2.cloudflarestorage.com")
	t.Setenv("TF_R2_BUCKET_PHOTOS", "test-photos")
	t.Setenv("TF_R2_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("TF_R2_SECRET_ACCESS_KEY", "test-secret-key")
}

func TestLoadRejectsPartialR2Configuration(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_R2_SECRET_ACCESS_KEY", "")
	_, err := Load()
	require.ErrorContains(t, err, "TF_R2_SECRET_ACCESS_KEY")
	require.NotContains(t, err.Error(), "test-access-key")
}

func TestLoadProductionWithoutR2KeepsLegacyPhotoStorage(t *testing.T) {
	setValidProductionEnvironment(t)
	t.Setenv("TF_R2_ENDPOINT", "")
	t.Setenv("TF_R2_BUCKET_PHOTOS", "")
	t.Setenv("TF_R2_ACCESS_KEY_ID", "")
	t.Setenv("TF_R2_SECRET_ACCESS_KEY", "")

	cfg, err := Load()
	require.NoError(t, err)
	require.Empty(t, cfg.R2BucketPhotos)
}

func setValidProductionEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("TF_R2_ENDPOINT", "https://account.r2.cloudflarestorage.com")
	t.Setenv("TF_R2_BUCKET_PHOTOS", "test-photos")
	t.Setenv("TF_R2_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("TF_R2_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow_app:runtime-database-secret-123@db:5432/tallerflow?sslmode=require")
	t.Setenv("TF_BOOTSTRAP_DATABASE_URL", "postgres://tallerflow_bootstrap:bootstrap-database-secret-123@db:5432/tallerflow?sslmode=require")
	t.Setenv("TF_SESSION_PEPPER", "production-session-pepper-at-least-32-bytes")
	t.Setenv("TF_ALLOWED_ORIGIN", "https://app.tallerflow.test")
	t.Setenv("TF_ENVIRONMENT", "production")
	t.Setenv("TF_PASSWORD_RESET_BASE_URL", "https://app.tallerflow.test")
	t.Setenv("TF_SMTP_REQUIRE_TLS", "true")
	t.Setenv("TF_CONSOLE_DATABASE_URL", "postgres://tallerflow_console:console-database-secret-123@db:5432/tallerflow?sslmode=require")
	t.Setenv("TF_CONSOLE_ORIGIN", "https://admin.tallerflow.test")
	t.Setenv("TF_CONSOLE_TOTP_KEY", "production-console-totp-key-at-least-32-bytes")
}

func TestLoadOpensSignupOnlyInDevelopmentUnlessConfigured(t *testing.T) {
	setValidEnvironment(t)
	cfg, err := Load()
	require.NoError(t, err)
	require.True(t, cfg.SignupEnabled, "development keeps open signup on for local testing")

	setValidProductionEnvironment(t)
	cfg, err = Load()
	require.NoError(t, err)
	require.False(t, cfg.SignupEnabled, "production stays closed until TF_SIGNUP_ENABLED is set")

	t.Setenv("TF_SIGNUP_ENABLED", "true")
	cfg, err = Load()
	require.NoError(t, err)
	require.True(t, cfg.SignupEnabled)

	t.Setenv("TF_SIGNUP_ENABLED", "maybe")
	_, err = Load()
	require.ErrorContains(t, err, "TF_SIGNUP_ENABLED is invalid")
}
