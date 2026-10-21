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
	BootstrapDatabaseURL string
	SessionPepper        string
	AllowedOrigin        string
	MobileOrigins        []string
	TrustedProxies       []string
	PasswordResetBaseURL string
	SMTPHost             string
	SMTPPort             string
	SMTPUsername         string
	SMTPPassword         string
	SMTPFromAddress      string
	SMTPRequireTLS       bool
	R2Endpoint           string
	R2BucketPhotos       string
	R2AccessKeyID        string
	R2SecretAccessKey    string
	ConsoleDatabaseURL   string
	ConsoleOrigin        string
	ConsoleTOTPKey       string
}

// Load reads and validates the API process configuration.
func Load() (Config, error) {
	cfg := Config{
		Environment:          valueOrDefault("TF_ENVIRONMENT", defaultEnvironment),
		HTTPAddress:          valueOrDefault("TF_HTTP_ADDRESS", defaultHTTPAddress),
		DatabaseURL:          os.Getenv("TF_DATABASE_URL"),
		BootstrapDatabaseURL: os.Getenv("TF_BOOTSTRAP_DATABASE_URL"),
		SessionPepper:        os.Getenv("TF_SESSION_PEPPER"),
		AllowedOrigin:        strings.TrimSpace(os.Getenv("TF_ALLOWED_ORIGIN")),
		SMTPHost:             valueOrDefault("TF_SMTP_HOST", "localhost"),
		SMTPPort:             valueOrDefault("TF_SMTP_PORT", "1025"),
		SMTPUsername:         strings.TrimSpace(os.Getenv("TF_SMTP_USER")),
		SMTPPassword:         os.Getenv("TF_SMTP_PASSWORD"),
		SMTPFromAddress:      valueOrDefault("TF_SMTP_FROM", "soporte@tallerflow.pe"),
		R2Endpoint:           strings.TrimSpace(os.Getenv("TF_R2_ENDPOINT")),
		R2BucketPhotos:       strings.TrimSpace(os.Getenv("TF_R2_BUCKET_PHOTOS")),
		R2AccessKeyID:        strings.TrimSpace(os.Getenv("TF_R2_ACCESS_KEY_ID")),
		R2SecretAccessKey:    os.Getenv("TF_R2_SECRET_ACCESS_KEY"),
		ConsoleDatabaseURL:   strings.TrimSpace(os.Getenv("TF_CONSOLE_DATABASE_URL")),
		ConsoleOrigin:        strings.TrimSpace(os.Getenv("TF_CONSOLE_ORIGIN")),
		ConsoleTOTPKey:       os.Getenv("TF_CONSOLE_TOTP_KEY"),
	}

	missing := make([]string, 0, 4)
	if strings.TrimSpace(cfg.DatabaseURL) == "" {
		missing = append(missing, "TF_DATABASE_URL")
	}
	if strings.TrimSpace(cfg.SessionPepper) == "" {
		missing = append(missing, "TF_SESSION_PEPPER")
	}
	if strings.TrimSpace(cfg.BootstrapDatabaseURL) == "" {
		missing = append(missing, "TF_BOOTSTRAP_DATABASE_URL")
	}
	if cfg.AllowedOrigin == "" {
		missing = append(missing, "TF_ALLOWED_ORIGIN")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	configuredR2 := cfg.R2Endpoint != "" || cfg.R2BucketPhotos != "" || cfg.R2AccessKeyID != "" || cfg.R2SecretAccessKey != ""
	if configuredR2 &&
		(cfg.R2Endpoint == "" || cfg.R2BucketPhotos == "" || cfg.R2AccessKeyID == "" || cfg.R2SecretAccessKey == "") {
		return Config{}, fmt.Errorf("TF_R2_ENDPOINT, TF_R2_BUCKET_PHOTOS, TF_R2_ACCESS_KEY_ID and TF_R2_SECRET_ACCESS_KEY are required together")
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
	if cfg.ConsoleOrigin == "" {
		cfg.ConsoleOrigin = cfg.AllowedOrigin
	}
	consoleOrigin, err := url.Parse(cfg.ConsoleOrigin)
	if err != nil || consoleOrigin.Host == "" || (consoleOrigin.Scheme != "http" && consoleOrigin.Scheme != "https") {
		return Config{}, fmt.Errorf("TF_CONSOLE_ORIGIN is invalid")
	}
	if cfg.Environment == "production" && consoleOrigin.Scheme != "https" {
		return Config{}, fmt.Errorf("TF_CONSOLE_ORIGIN is invalid")
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
	if cfg.Environment == "production" {
		if err := validateProductionSecrets(cfg); err != nil {
			return Config{}, err
		}
	}

	cfg.TrustedProxies, err = trustedProxies(cfg.Environment, os.Getenv("TF_TRUSTED_PROXIES"))
	if err != nil {
		return Config{}, err
	}

	cfg.MobileOrigins, err = mobileOrigins(cfg.Environment, os.Getenv("TF_MOBILE_ORIGINS"))
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// mobileOrigins lists the WebView origins of the Android (and later iOS) app.
// Capacitor serves the bundled web app from https://localhost on Android and
// capacitor://localhost on iOS; plain http is accepted only outside production.
func mobileOrigins(environment, configured string) ([]string, error) {
	configured = strings.TrimSpace(configured)
	if configured == "" {
		if environment == "production" {
			return []string{"https://localhost", "capacitor://localhost"}, nil
		}
		return []string{"http://localhost", "https://localhost", "capacitor://localhost"}, nil
	}

	origins := strings.Split(configured, ",")
	for index := range origins {
		origin := strings.ToLower(strings.TrimSpace(origins[index]))
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
			return nil, fmt.Errorf("TF_MOBILE_ORIGINS is invalid")
		}
		switch parsed.Scheme {
		case "https", "capacitor":
		case "http":
			if environment == "production" {
				return nil, fmt.Errorf("TF_MOBILE_ORIGINS is invalid")
			}
		default:
			return nil, fmt.Errorf("TF_MOBILE_ORIGINS is invalid")
		}
		origins[index] = origin
	}
	return origins, nil
}

func validateProductionSecrets(cfg Config) error {
	if len(cfg.SessionPepper) < 32 || cfg.SessionPepper == "local-session-pepper-change-me" {
		return fmt.Errorf("TF_SESSION_PEPPER is insecure for production")
	}
	if err := validateProductionDatabaseURL("TF_DATABASE_URL", cfg.DatabaseURL, "tallerflow_app"); err != nil {
		return err
	}
	if err := validateProductionDatabaseURL("TF_BOOTSTRAP_DATABASE_URL", cfg.BootstrapDatabaseURL, "tallerflow_bootstrap"); err != nil {
		return err
	}
	if cfg.DatabaseURL == cfg.BootstrapDatabaseURL {
		return fmt.Errorf("production database roles must be separate")
	}
	if err := validateProductionDatabaseURL("TF_CONSOLE_DATABASE_URL", cfg.ConsoleDatabaseURL, "tallerflow_console"); err != nil {
		return err
	}
	if len(cfg.ConsoleTOTPKey) < 32 || strings.Contains(cfg.ConsoleTOTPKey, "change-me") {
		return fmt.Errorf("TF_CONSOLE_TOTP_KEY is insecure for production")
	}
	if cfg.ConsoleDatabaseURL == cfg.DatabaseURL || cfg.ConsoleDatabaseURL == cfg.BootstrapDatabaseURL {
		return fmt.Errorf("production database roles must be separate")
	}
	return nil
}

func validateProductionDatabaseURL(name, rawURL, expectedUser string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" || parsed.User == nil {
		return fmt.Errorf("%s is invalid for production", name)
	}
	password, hasPassword := parsed.User.Password()
	if parsed.User.Username() != expectedUser || !hasPassword || len(password) < 16 || strings.Contains(password, "change-me") {
		return fmt.Errorf("%s is insecure for production", name)
	}
	if strings.EqualFold(parsed.Query().Get("sslmode"), "disable") {
		return fmt.Errorf("%s must not disable TLS in production", name)
	}
	return nil
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
