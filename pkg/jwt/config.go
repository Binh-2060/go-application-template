package jwt

import (
	"fmt"
	"os"
	"time"
)

// Config holds the knobs needed to construct a Manager.
type Config struct {
	Secret string
	Issuer string
	TTL    time.Duration
}

// RSAConfig holds the knobs needed to construct an RSAManager.
type RSAConfig struct {
	PrivateKeyPEM string // empty builds a verify-only Manager
	PublicKeyPEM  string
	Issuer        string
	TTL           time.Duration
}

/*
Read JWT configuration from environment variables.

Recognised vars: JWT_SECRET (required, no default — Manager rejects an empty
secret), JWT_ISSUER (default API_NAME, falling back to
"go-fiber-api-template"), JWT_TTL (default 24h).
*/
func ConfigFromEnv() Config {
	return Config{
		Secret: os.Getenv("JWT_SECRET"),
		Issuer: envOr("JWT_ISSUER", envOr("API_NAME", "go-fiber-api-template")),
		TTL:    envDurationOr("JWT_TTL", 24*time.Hour),
	}
}

/*
Read RSA JWT configuration from environment variables.

Recognised vars: JWT_RSA_PRIVATE_KEY_PATH (path to a PEM-encoded RSA private
key, optional — omit to build a verify-only Manager), JWT_RSA_PUBLIC_KEY_PATH
(path to the matching PEM-encoded public key, required), JWT_ISSUER, JWT_TTL
(same defaults as ConfigFromEnv).

Keys are read from files rather than inlined in .env so PEM material — which
spans multiple lines — doesn't need escaping into a single env var, and
doesn't get dumped by `env`/process-listing tools the way an env var would.
*/
func RSAConfigFromEnv() (RSAConfig, error) {
	cfg := RSAConfig{
		Issuer: envOr("JWT_ISSUER", envOr("API_NAME", "go-fiber-api-template")),
		TTL:    envDurationOr("JWT_TTL", 24*time.Hour),
	}

	if path := os.Getenv("JWT_RSA_PRIVATE_KEY_PATH"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return RSAConfig{}, fmt.Errorf("jwt: read private key: %w", err)
		}
		cfg.PrivateKeyPEM = string(b)
	}

	if path := os.Getenv("JWT_RSA_PUBLIC_KEY_PATH"); path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return RSAConfig{}, fmt.Errorf("jwt: read public key: %w", err)
		}
		cfg.PublicKeyPEM = string(b)
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	v, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return fallback
	}
	return v
}
