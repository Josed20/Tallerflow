package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultEnvironment = "development"
	defaultHTTPAddress = ":8080"
)

// Config contains the process configuration read from TF_* environment
// variables. Secret values must never be included in returned errors.
type Config struct {
	Environment   string
	HTTPAddress   string
	DatabaseURL   string
	SessionPepper string
	AllowedOrigin string
}

// Load reads and validates the API process configuration.
func Load() (Config, error) {
	cfg := Config{
		Environment:   valueOrDefault("TF_ENVIRONMENT", defaultEnvironment),
		HTTPAddress:   valueOrDefault("TF_HTTP_ADDRESS", defaultHTTPAddress),
		DatabaseURL:   os.Getenv("TF_DATABASE_URL"),
		SessionPepper: os.Getenv("TF_SESSION_PEPPER"),
		AllowedOrigin: os.Getenv("TF_ALLOWED_ORIGIN"),
	}

	missing := make([]string, 0, 2)
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		missing = append(missing, "TF_DATABASE_URL")
	}
	if strings.TrimSpace(cfg.SessionPepper) == "" {
		missing = append(missing, "TF_SESSION_PEPPER")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
