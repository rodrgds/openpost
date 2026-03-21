package config

import (
	"os"
)

type Config struct {
	Port          string
	DatabasePath  string
	JWTSecret     string
	EncryptionKey string
	FrontendURL   string

	TwitterClientID     string
	TwitterClientSecret string

	MastodonClientID     string
	MastodonClientSecret string
	MastodonRedirectURI  string
}

func Load() *Config {
	return &Config{
		Port:          getEnv("OPENPOST_PORT", "8080"),
		DatabasePath:  getEnv("OPENPOST_DB_PATH", "file:openpost.db?cache=shared&mode=rwc"),
		JWTSecret:     getEnv("JWT_SECRET", "development-jwt-secret-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "super-secret-32-byte-master-key-here"),
		FrontendURL:   getEnv("OPENPOST_FRONTEND_URL", "http://localhost:5173"),

		TwitterClientID:     getEnv("TWITTER_CLIENT_ID", ""),
		TwitterClientSecret: getEnv("TWITTER_CLIENT_SECRET", ""),

		MastodonClientID:     getEnv("MASTODON_CLIENT_ID", ""),
		MastodonClientSecret: getEnv("MASTODON_CLIENT_SECRET", ""),
		MastodonRedirectURI:  getEnv("MASTODON_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/mastodon/callback"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
