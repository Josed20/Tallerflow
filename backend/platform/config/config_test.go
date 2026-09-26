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
	t.Setenv("TF_SESSION_PEPPER", "test-session-pepper")
	t.Setenv("TF_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("TF_HTTP_ADDRESS", "")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.HTTPAddress)
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

	t.Setenv("TF_ENVIRONMENT", "production")
	t.Setenv("TF_ALLOWED_ORIGIN", "https://app.tallerflow.test")
	production, err := Load()
	require.NoError(t, err)
	require.Equal(t, []string{"172.16.0.0/12"}, production.TrustedProxies)
}

func TestLoadRejectsInvalidTrustedProxyWithoutLeakingValue(t *testing.T) {
	setValidEnvironment(t)
	t.Setenv("TF_TRUSTED_PROXIES", "not-a-network-secret")

	_, err := Load()

	require.ErrorContains(t, err, "TF_TRUSTED_PROXIES")
	require.False(t, strings.Contains(err.Error(), "not-a-network-secret"))
}

func setValidEnvironment(t *testing.T) {
	t.Helper()
	t.Setenv("TF_DATABASE_URL", "postgres://tallerflow:password@localhost:5432/tallerflow")
	t.Setenv("TF_SESSION_PEPPER", "test-session-pepper")
	t.Setenv("TF_ALLOWED_ORIGIN", "http://localhost:5173")
	t.Setenv("TF_ENVIRONMENT", "development")
}
