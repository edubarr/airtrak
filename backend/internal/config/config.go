package config

import (
	"fmt"
	"os"
	"time"
)

const defaultJWTSecret = "change-me-in-dev"

type Config struct {
	AppEnv         string
	DatabaseURL    string
	APIAddr        string
	JWTSecret      string
	DefaultLocale  string
	AccessTokenTTL time.Duration
}

func Load() (Config, error) {
	ttl, err := time.ParseDuration(getenv("AUTH_ACCESS_TOKEN_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse AUTH_ACCESS_TOKEN_TTL: %w", err)
	}

	cfg := Config{
		AppEnv:         getenv("APP_ENV", "development"),
		DatabaseURL:    getenv("APP_DATABASE_URL", "postgres://airtrak:airtrak@localhost:5432/airtrak?sslmode=disable"),
		APIAddr:        getenv("API_ADDR", ":8080"),
		JWTSecret:      getenv("AUTH_JWT_SECRET", defaultJWTSecret),
		AccessTokenTTL: ttl,
		DefaultLocale:  getenv("APP_DEFAULT_LOCALE", "pt-BR"),
	}

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("APP_DATABASE_URL is required")
	}

	if cfg.JWTSecret == "" {
		return Config{}, fmt.Errorf("AUTH_JWT_SECRET is required")
	}

	if cfg.AppEnv == "production" && cfg.JWTSecret == defaultJWTSecret {
		return Config{}, fmt.Errorf("AUTH_JWT_SECRET must be set in production")
	}

	if cfg.AccessTokenTTL <= 0 {
		return Config{}, fmt.Errorf("AUTH_ACCESS_TOKEN_TTL must be positive")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
