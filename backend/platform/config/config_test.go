package config

import (
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
	t.Setenv("TF_HTTP_ADDRESS", "")

	cfg, err := Load()

	require.NoError(t, err)
	require.Equal(t, ":8080", cfg.HTTPAddress)
}
