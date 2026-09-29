package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	defaultEnvironment = "development"
	defaultHTTPAddress = ":8080"
)

// Config contains the process configuration read from TF_* environment
// variables. Secret values must never be included in returned errors.
type Config struct {
	Environment          string
	HTTPAddress          string
	DatabaseURL          string
	SessionPepper        string
	AllowedOrigin        string
	TrustedProxies       []string
	PasswordResetBaseURL string
	SMTPHost             string
	SMTPPort             string
	SMTPUsername         string
	SMTPPassword         string
	SMTPFromAddress      string
	SMTPRequireTLS       bool
}

// Load reads and validates the API process configuration.
func Load() (Config, error) {
	cfg := Config{
		Environment:     valueOrDefault("TF_ENVIRONMENT", defaultEnvironment),
		HTTPAddress:     valueOrDefault("TF_HTTP_ADDRESS", defaultHTTPAddress),
		DatabaseURL:     os.Getenv("TF_DATABASE_URL"),
		SessionPepper:   os.Getenv("TF_SESSION_PEPPER"),
		AllowedOrigin:   strings.TrimSpace(os.Getenv("TF_ALLOWED_ORIGIN")),
		SMTPHost:        valueOrDefault("TF_SMTP_HOST", "localhost"),
		SMTPPort:        valueOrDefault("TF_SMTP_PORT", "1025"),
		SMTPUsername:    strings.TrimSpace(os.Getenv("TF_SMTP_USER")),
		SMTPPassword:    os.Getenv("TF_SMTP_PASSWORD"),
		SMTPFromAddress: valueOrDefault("TF_SMTP_FROM", "soporte@tallerflow.pe"),
	}

	missing := make([]string, 0, 3)
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		missing = append(missing, "TF_DATABASE_URL")
	}
	if strings.TrimSpace(cfg.SessionPepper) == "" {
		missing = append(missing, "TF_SESSION_PEPPER")
	}
	if cfg.AllowedOrigin == "" {
		missing = append(missing, "TF_ALLOWED_ORIGIN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if !validEnvironment(cfg.Environment) {
		return Config{}, fmt.Errorf("TF_ENVIRONMENT is invalid")
	}
	origin, err := url.Parse(cfg.AllowedOrigin)
	if err != nil || origin.Host == "" || (origin.Scheme != "http" && origin.Scheme != "https") {
		return Config{}, fmt.Errorf("TF_ALLOWED_ORIGIN is invalid")
	}
	if cfg.Environment == "production" && origin.Scheme != "https" {
		return Config{}, fmt.Errorf("TF_ALLOWED_ORIGIN is invalid")
	}
	cfg.PasswordResetBaseURL = valueOrDefault("TF_PASSWORD_RESET_BASE_URL", cfg.AllowedOrigin)
	resetURL, err := url.Parse(cfg.PasswordResetBaseURL)
	if err != nil || resetURL.Host == "" || (resetURL.Scheme != "http" && resetURL.Scheme != "https") {
		return Config{}, fmt.Errorf("TF_PASSWORD_RESET_BASE_URL is invalid")
	}
	if cfg.Environment == "production" && resetURL.Scheme != "https" {
		return Config{}, fmt.Errorf("TF_PASSWORD_RESET_BASE_URL is invalid")
	}

	cfg.SMTPRequireTLS = cfg.Environment == "production"
	if configuredTLS := strings.TrimSpace(os.Getenv("TF_SMTP_REQUIRE_TLS")); configuredTLS != "" {
		cfg.SMTPRequireTLS, err = strconv.ParseBool(configuredTLS)
		if err != nil {
			return Config{}, fmt.Errorf("TF_SMTP_REQUIRE_TLS is invalid")
		}
	}
	if cfg.Environment == "production" && !cfg.SMTPRequireTLS {
		return Config{}, fmt.Errorf("TF_SMTP_REQUIRE_TLS is invalid")
	}

	cfg.TrustedProxies, err = trustedProxies(cfg.Environment, os.Getenv("TF_TRUSTED_PROXIES"))
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func validEnvironment(environment string) bool {
	switch environment {
	case "development", "test", "production":
		return true
	default:
		return false
	}
}

func trustedProxies(environment, configured string) ([]string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		if environment == "production" {
			return []string{"172.16.0.0/12"}, nil
		}
		return nil, nil
	}

	proxies := strings.Split(configured, ",")
	for index := range proxies {
		proxy := strings.TrimSpace(proxies[index])
		if proxy == "" || !validProxy(proxy) {
			return nil, fmt.Errorf("TF_TRUSTED_PROXIES is invalid")
		}
		proxies[index] = proxy
	}
	return proxies, nil
}

func validProxy(value string) bool {
	if net.ParseIP(value) != nil {
		return true
	}
	_, _, err := net.ParseCIDR(value)
	return err == nil
}

func valueOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
