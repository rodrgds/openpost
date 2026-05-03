package config

import (
	"encoding/json"
	"log"
	"net/url"
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
	Port                 string
	DatabasePath         string
	JWTSecret            string
	EncryptionKey        string
	DisableRegistrations bool
	FrontendURL          string
	PublicURL            string
	CORSOrigins          []string
	WebAuthnRPID         string

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
		Port:                 os.Getenv("OPENPOST_PORT"),
		DatabasePath:         os.Getenv("OPENPOST_DATABASE_PATH"),
		JWTSecret:            os.Getenv("OPENPOST_JWT_SECRET"),
		EncryptionKey:        os.Getenv("OPENPOST_ENCRYPTION_KEY"),
		DisableRegistrations: getEnvBool("OPENPOST_DISABLE_REGISTRATIONS", false),
		FrontendURL:          os.Getenv("OPENPOST_APP_URL"),
		PublicURL:            os.Getenv("OPENPOST_PUBLIC_URL"),

		TwitterClientID:     os.Getenv("X_CLIENT_ID"),
		TwitterClientSecret: os.Getenv("X_CLIENT_SECRET"),
		TwitterRedirectURI:  os.Getenv("X_REDIRECT_URI"),

		MastodonRedirectURI: getEnvDefault("MASTODON_REDIRECT_URI", "http://localhost:5173/api/v1/accounts/mastodon/callback"),

		LinkedInClientID:             os.Getenv("LINKEDIN_CLIENT_ID"),
		LinkedInClientSecret:         os.Getenv("LINKEDIN_CLIENT_SECRET"),
		LinkedInRedirectURI:          os.Getenv("LINKEDIN_REDIRECT_URI"),
		DisableLinkedInThreadReplies: getEnvBool("LINKEDIN_DISABLE_THREAD_REPLIES", false),

		ThreadsClientID:     os.Getenv("THREADS_CLIENT_ID"),
		ThreadsClientSecret: os.Getenv("THREADS_CLIENT_SECRET"),
		ThreadsRedirectURI:  os.Getenv("THREADS_REDIRECT_URI"),

		MediaPath: getEnvDefault("OPENPOST_MEDIA_PATH", "./media"),
		MediaURL:  getEnvDefault("OPENPOST_MEDIA_URL", "/media"),
	}

	if cfg.PublicURL == "" {
		cfg.PublicURL = cfg.FrontendURL
	}
	if parsed, err := url.Parse(cfg.PublicURL); err == nil && parsed.Hostname() != "" {
		cfg.WebAuthnRPID = parsed.Hostname()
	} else {
		cfg.WebAuthnRPID = "localhost"
	}

	if raw := os.Getenv("MASTODON_SERVERS"); raw != "" {
		var servers []MastodonServerConfig
		if err := json.Unmarshal([]byte(raw), &servers); err != nil {
			log.Printf("WARNING: failed to parse MASTODON_SERVERS JSON: %v", err)
		} else {
			cfg.MastodonServers = servers
		}
	}

	// Build CORS origins list
	corsOrigins := []string{cfg.FrontendURL, "http://localhost:5173"}
	if extra := os.Getenv("OPENPOST_EXTRA_CORS_ORIGINS"); extra != "" {
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

func getEnvDefault(key, fallback string) string {
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

func Init() {
	// Validate critical security config on startup
	jwtSecret := os.Getenv("OPENPOST_JWT_SECRET")
	encryptionKey := os.Getenv("OPENPOST_ENCRYPTION_KEY")

	isProduction := os.Getenv("OPENPOST_ENV") == "production" ||
		os.Getenv("OPENPOST_ENV") == "prod" ||
		os.Getenv("GIN_MODE") == "release"

	if isProduction {
		if jwtSecret == "" {
			log.Fatal("FATAL: OPENPOST_JWT_SECRET is required in production. Set OPENPOST_ENV=production to enable this check.")
		}
		if len(jwtSecret) < 32 {
			log.Printf("FATAL: OPENPOST_JWT_SECRET must be at least 32 characters in production (got %d)", len(jwtSecret))
		}
		if encryptionKey == "" {
			log.Fatal("FATAL: OPENPOST_ENCRYPTION_KEY is required in production. Set OPENPOST_ENV=production to enable this check.")
		}
		if len(encryptionKey) < 32 {
			log.Fatalf("FATAL: OPENPOST_ENCRYPTION_KEY must be at least 32 characters in production (got %d)", len(encryptionKey))
		}
	} else {
		// Warn in development if using defaults
		if jwtSecret == "" {
			log.Println("WARNING: OPENPOST_JWT_SECRET is not set. Set OPENPOST_JWT_SECRET in .env for production.")
		}
		if encryptionKey == "" {
			log.Println("WARNING: OPENPOST_ENCRYPTION_KEY is not set. Set OPENPOST_ENCRYPTION_KEY in .env for production.")
		}
	}
}
