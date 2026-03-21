package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/openpost/backend/internal/api/handlers"
	apimiddleware "github.com/openpost/backend/internal/api/middleware"
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
	// Load .env file if it exists
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
	maAuth := oauth.NewMastodonOAuth(
		cfg.MastodonClientID,
		cfg.MastodonClientSecret,
		cfg.MastodonRedirectURI,
	)

	publishSvc := publisher.NewService(db, tokenManager)

	worker := queue.NewWorker(db, "worker-1", 5*time.Second, publishSvc)
	go worker.Start(context.Background())

	authHandler := handlers.NewAuthHandler(db, authService)
	workspaceHandler := handlers.NewWorkspaceHandler(db)
	postHandler := handlers.NewPostHandler(db)
	oauthHandler := handlers.NewOAuthHandler(db, tokenEncryptor, twAuth, maAuth)

	api := e.Group("/api/v1")

	// Public auth routes
	api.POST("/auth/register", authHandler.Register)
	api.POST("/auth/login", authHandler.Login)

	// OAuth callback routes (public - called by OAuth providers)
	api.GET("/accounts/:platform/callback", oauthHandler.Callback)
	api.POST("/accounts/mastodon/exchange", oauthHandler.ExchangeCode)

	// Protected routes
	protected := api.Group("")
	protected.Use(apimiddleware.AuthMiddleware(authService))

	protected.GET("/auth/me", authHandler.Me)
	protected.POST("/workspaces", workspaceHandler.CreateWorkspace)
	protected.GET("/workspaces", workspaceHandler.ListWorkspaces)
	protected.POST("/posts", postHandler.CreatePost)
	protected.GET("/posts", postHandler.ListPosts)
	protected.GET("/posts/schedule-overview", postHandler.GetScheduleOverview)
	// Returns OAuth URL as JSON (frontend redirects browser to it)
	protected.GET("/accounts/:platform/auth-url", oauthHandler.GetAuthURL)
	protected.GET("/accounts", oauthHandler.ListAccounts)

	api.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	RegisterSpaRoutes(e)

	log.Println("Starting OpenPost on :" + cfg.Port)
	log.Fatal(e.Start(":" + cfg.Port))
}
