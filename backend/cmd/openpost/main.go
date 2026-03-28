package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/openpost/backend/internal/api/handlers"
	"github.com/openpost/backend/internal/config"
	"github.com/openpost/backend/internal/database"
	"github.com/openpost/backend/internal/queue"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/openpost/backend/internal/services/oauth"
	"github.com/openpost/backend/internal/services/publisher"
	"github.com/openpost/backend/internal/services/tokenmanager"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{cfg.FrontendURL, "http://localhost:5173", "http://localhost:8080"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	if err := database.CreateSchema(db); err != nil {
		log.Printf("CreateSchema error (already exists?): %v", err)
	}

	tokenEncryptor := crypto.NewTokenEncryptor(cfg.EncryptionKey)
	authService := auth.NewService(cfg.JWTSecret)
	tokenManager := tokenmanager.NewTokenManager(db, tokenEncryptor)

	twAuth := oauth.NewTwitterOAuth(
		cfg.TwitterClientID,
		cfg.TwitterClientSecret,
		"http://localhost:8080/api/v1/accounts/x/callback",
	)

	mastodonServers := make(map[string]*oauth.MastodonOAuth)
	for _, server := range cfg.MastodonServers {
		mastodonServers[server.Name] = oauth.NewMastodonOAuth(
			server.ClientID,
			server.ClientSecret,
			cfg.MastodonRedirectURI,
			server.InstanceURL,
		)
		log.Printf("Registered Mastodon server: %s (%s)", server.Name, server.InstanceURL)
	}

	var blueskyAuth *oauth.BlueskyOAuth
	// Bluesky uses app passwords, no OAuth setup needed
	blueskyAuth = oauth.NewBlueskyOAuth("") // uses https://bsky.social by default

	var linkedinAuth *oauth.LinkedInOAuth
	if cfg.LinkedInClientID != "" {
		linkedinAuth = oauth.NewLinkedInOAuth(
			cfg.LinkedInClientID,
			cfg.LinkedInClientSecret,
			"http://localhost:8080/api/v1/accounts/linkedin/callback",
		)
	}

	var threadsAuth *oauth.ThreadsOAuth
	if cfg.ThreadsClientID != "" {
		threadsAuth = oauth.NewThreadsOAuth(
			cfg.ThreadsClientID,
			cfg.ThreadsClientSecret,
			"http://localhost:8080/api/v1/accounts/threads/callback",
		)
	}

	publishSvc := publisher.NewService(db, tokenManager)
	if blueskyAuth != nil {
		publishSvc.SetBlueskyOAuth(blueskyAuth)
	}
	if linkedinAuth != nil {
		publishSvc.SetLinkedInOAuth(linkedinAuth)
		tokenManager.SetLinkedInOAuth(linkedinAuth)
	}
	if threadsAuth != nil {
		publishSvc.SetThreadsOAuth(threadsAuth)
		tokenManager.SetThreadsOAuth(threadsAuth)
	}
	if twAuth != nil {
		tokenManager.SetTwitterOAuth(twAuth)
	}
	tokenManager.SetMastodonOAuth(mastodonServers)

	worker := queue.NewWorker(db, "worker-1", 5*time.Second, publishSvc)
	go worker.Start(context.Background())

	// --- Huma setup ---
	apiGroup := e.Group("/api/v1")
	humaConfig := huma.DefaultConfig("OpenPost API", "1.0.0")
	api := humaecho.NewWithGroup(e, apiGroup, humaConfig)

	// Serve OpenAPI spec
	e.GET("/openapi.json", func(c echo.Context) error {
		spec := api.OpenAPI()
		data, err := json.Marshal(spec)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal spec"})
		}
		return c.Blob(http.StatusOK, "application/json", data)
	})

	// Register handlers
	authHandler := handlers.NewAuthHandler(db, authService)
	authHandler.Register(api)
	authHandler.Login(api)
	authHandler.Me(api)

	workspaceHandler := handlers.NewWorkspaceHandler(db, authService)
	workspaceHandler.CreateWorkspace(api)
	workspaceHandler.ListWorkspaces(api)

	postHandler := handlers.NewPostHandler(db, authService)
	postHandler.CreatePost(api)
	postHandler.ListPosts(api)
	postHandler.GetScheduleOverview(api)

	oauthHandler := handlers.NewOAuthHandler(db, tokenEncryptor, twAuth, mastodonServers, blueskyAuth, linkedinAuth, threadsAuth, authService)
	oauthHandler.ListMastodonServers(api)
	oauthHandler.GetAuthURL(api)
	oauthHandler.Callback(api)
	oauthHandler.ExchangeCode(api)
	oauthHandler.BlueskyLogin(api)
	oauthHandler.ListAccounts(api)

	// Health check (Huma-registered for OpenAPI docs)
	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Tags:        []string{"System"},
	}, func(ctx context.Context, input *struct{}) (*struct {
		Body struct {
			Status string `json:"status" doc:"Health status"`
		}
	}, error) {
		resp := &struct {
			Body struct {
				Status string `json:"status" doc:"Health status"`
			}
		}{}
		resp.Body.Status = "ok"
		return resp, nil
	})

	// SPA routes (must be last - catches all unmatched routes)
	RegisterSpaRoutes(e)

	log.Println("Starting OpenPost on :" + cfg.Port)
	log.Println("OpenAPI spec available at http://localhost:" + cfg.Port + "/openapi.json")
	log.Fatal(e.Start(":" + cfg.Port))
}
