package config

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"
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
	CORSOrigins   []string

	TwitterClientID     string
	TwitterClientSecret string
	TwitterRedirectURI  string

	MastodonRedirectURI string
	MastodonServers     []MastodonServerConfig

	LinkedInClientID             string
	LinkedInClientSecret         string
	LinkedInRedirectURI          string
	DisableLinkedInThreadReplies bool

	ThreadsClientID     string
	ThreadsClientSecret string
	ThreadsRedirectURI  string

	MediaPath string
	MediaURL  string
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
		TwitterRedirectURI:  getEnv("TWITTER_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/x/callback"),

		MastodonRedirectURI: getEnv("MASTODON_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/mastodon/callback"),

		LinkedInClientID:             getEnv("LINKEDIN_CLIENT_ID", ""),
		LinkedInClientSecret:         getEnv("LINKEDIN_CLIENT_SECRET", ""),
		LinkedInRedirectURI:          getEnv("LINKEDIN_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/linkedin/callback"),
		DisableLinkedInThreadReplies: getEnvBool("OPENPOST_DISABLE_LINKEDIN_THREAD_REPLIES", false),

		ThreadsClientID:     getEnv("THREADS_CLIENT_ID", ""),
		ThreadsClientSecret: getEnv("THREADS_CLIENT_SECRET", ""),
		ThreadsRedirectURI:  getEnv("THREADS_REDIRECT_URI", "http://localhost:8080/api/v1/accounts/threads/callback"),

		MediaPath: getEnv("OPENPOST_MEDIA_PATH", "./media"),
		MediaURL:  getEnv("OPENPOST_MEDIA_URL", "/media"),
	}

	if raw := getEnv("MASTODON_SERVERS", ""); raw != "" {
		var servers []MastodonServerConfig
		if err := json.Unmarshal([]byte(raw), &servers); err != nil {
			log.Printf("WARNING: failed to parse MASTODON_SERVERS JSON: %v", err)
		} else {
			cfg.MastodonServers = servers
		}
	}

	// Build CORS origins list
	corsOrigins := []string{cfg.FrontendURL, "http://localhost:5173", "http://localhost:8080"}
	if extra := getEnv("OPENPOST_CORS_EXTRA_ORIGINS", ""); extra != "" {
		for _, origin := range strings.Split(extra, ",") {
			trimmed := strings.TrimSpace(origin)
			if trimmed != "" {
				corsOrigins = append(corsOrigins, trimmed)
			}
		}
	}
	// Always allow Capacitor origins
	corsOrigins = append(corsOrigins, "capacitor://localhost", "http://localhost", "https://localhost")
	cfg.CORSOrigins = corsOrigins

	return cfg
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("WARNING: invalid boolean for %s=%q, using default %t", key, value, fallback)
		return fallback
	}

	return parsed
}

func init() {
	// Validate critical security config on startup
	jwtSecret := os.Getenv("JWT_SECRET")
	encryptionKey := os.Getenv("ENCRYPTION_KEY")

	isProduction := os.Getenv("OPENPOST_ENV") == "production" ||
		os.Getenv("OPENPOST_ENV") == "prod" ||
		os.Getenv("GIN_MODE") == "release"

	if isProduction {
		if jwtSecret == "" {
			log.Fatal("FATAL: JWT_SECRET is required in production. Set OPENPOST_ENV=production to enable this check.")
		}
		if len(jwtSecret) < 32 {
			log.Printf("FATAL: JWT_SECRET must be at least 32 characters in production (got %d)", len(jwtSecret))
		}
		if encryptionKey == "" {
			log.Fatal("FATAL: ENCRYPTION_KEY is required in production. Set OPENPOST_ENV=production to enable this check.")
		}
		if len(encryptionKey) < 32 {
			log.Fatalf("FATAL: ENCRYPTION_KEY must be at least 32 characters in production (got %d)", len(encryptionKey))
		}
	} else {
		// Warn in development if using defaults
		if jwtSecret == "development-jwt-secret-change-in-production" {
			log.Println("WARNING: Using default JWT_SECRET. Set JWT_SECRET in .env for production.")
		}
		if encryptionKey == "super-secret-32-byte-master-key-here" {
			log.Println("WARNING: Using default ENCRYPTION_KEY. Set ENCRYPTION_KEY in .env for production.")
		}
	}
}
