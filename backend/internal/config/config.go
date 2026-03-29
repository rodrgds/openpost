package config

import (
	"encoding/json"
	"log"
	"os"
)

type MastodonServerConfig struct {
	Name         string `json:"name"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	InstanceURL  string `json:"instance_url"`
}

type Config struct {
	Port          string
	DatabasePath  string
	JWTSecret     string
	EncryptionKey string
	FrontendURL   string

	TwitterClientID     string
	TwitterClientSecret string

	MastodonRedirectURI string
	MastodonServers     []MastodonServerConfig

	LinkedInClientID     string
	LinkedInClientSecret string

	ThreadsClientID     string
	ThreadsClientSecret string
	ThreadsRedirectURI  string
}

func Load() *Config {
	cfg := &Config{
		Port:          getEnv("OPENPOST_PORT", "8080"),
		DatabasePath:  getEnv("OPENPOST_DB_PATH", "file:openpost.db?cache=shared&mode=rwc"),
		JWTSecret:     getEnv("JWT_SECRET", "development-jwt-secret-change-in-production"),
		EncryptionKey: getEnv("ENCRYPTION_KEY", "super-secret-32-byte-master-key-here"),
		FrontendURL:   getEnv("OPENPOST_FRONTEND_URL", "http://localhost:5173"),

		TwitterClientID:     getEnv("TWITTER_CLIENT_ID", ""),
		TwitterClientSecret: getEnv("TWITTER_CLIENT_SECRET", ""),

		MastodonRedirectURI: getEnv("MASTODON_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/mastodon/callback"),

		LinkedInClientID:     getEnv("LINKEDIN_CLIENT_ID", ""),
		LinkedInClientSecret: getEnv("LINKEDIN_CLIENT_SECRET", ""),

		ThreadsClientID:     getEnv("THREADS_CLIENT_ID", ""),
		ThreadsClientSecret: getEnv("THREADS_CLIENT_SECRET", ""),
		ThreadsRedirectURI:  getEnv("THREADS_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/threads/callback"),
	}

	if raw := getEnv("MASTODON_SERVERS", ""); raw != "" {
		var servers []MastodonServerConfig
		if err := json.Unmarshal([]byte(raw), &servers); err != nil {
			log.Printf("WARNING: failed to parse MASTODON_SERVERS JSON: %v", err)
		} else {
			cfg.MastodonServers = servers
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
