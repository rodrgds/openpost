package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humaecho"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/openpost/backend/internal/api/handlers"
	"github.com/openpost/backend/internal/config"
	"github.com/openpost/backend/internal/database"
	"github.com/openpost/backend/internal/platform"
	"github.com/openpost/backend/internal/queue"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/openpost/backend/internal/services/mediastore"
	"github.com/openpost/backend/internal/services/publisher"
	"github.com/openpost/backend/internal/services/tokenmanager"
)

//nolint:gocyclo
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()
	config.Init()

	e := echo.New()
	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
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
	publishSvc := publisher.NewService(db, tokenManager)
	publishSvc.SetDisableLinkedInThreadReplies(cfg.DisableLinkedInThreadReplies)
	if cfg.MediaURL != "" && !strings.HasPrefix(cfg.MediaURL, "/") {
		publishSvc.SetPublicMediaURL(cfg.MediaURL)
	}

	providers := make(map[string]platform.Adapter)

	if cfg.TwitterClientID != "" {
		xAdapter := platform.NewXAdapter(
			cfg.TwitterClientID,
			cfg.TwitterClientSecret,
			cfg.TwitterRedirectURI,
		)
		providers["x"] = xAdapter
		log.Println("Registered X/Twitter adapter")
	}

	for _, server := range cfg.MastodonServers {
		mastodonAdapter := platform.NewMastodonAdapter(
			server.ClientID,
			server.ClientSecret,
			cfg.MastodonRedirectURI,
			server.InstanceURL,
		)
		providers["mastodon:"+server.Name] = mastodonAdapter
		log.Printf("Registered Mastodon adapter: %s (%s)", server.Name, server.InstanceURL)
	}

	blueskyAdapter := platform.NewBlueskyAdapter("")
	providers["bluesky"] = blueskyAdapter
	log.Println("Registered Bluesky adapter")

	if cfg.LinkedInClientID != "" {
		linkedinAdapter := platform.NewLinkedInAdapter(
			cfg.LinkedInClientID,
			cfg.LinkedInClientSecret,
			cfg.LinkedInRedirectURI,
			cfg.DisableLinkedInThreadReplies,
		)
		providers["linkedin"] = linkedinAdapter
		log.Println("Registered LinkedIn adapter")
	}

	if cfg.ThreadsClientID != "" {
		threadsAdapter := platform.NewThreadsAdapter(
			cfg.ThreadsClientID,
			cfg.ThreadsClientSecret,
			cfg.ThreadsRedirectURI,
		)
		providers["threads"] = threadsAdapter
		log.Println("Registered Threads adapter")
	}

	for name, adapter := range providers {
		tokenManager.SetProvider(name, adapter)
		publishSvc.SetProvider(name, adapter)
	}

	storage := mediastore.NewLocalStorage(cfg.MediaPath, cfg.MediaURL)
	if err := os.MkdirAll(filepath.Clean(cfg.MediaPath), 0755); err != nil {
		log.Printf("Warning: could not create media directory %s: %v", cfg.MediaPath, err)
	}
	mediaHandler := handlers.NewMediaHandler(db, storage, authService)

	worker := queue.NewWorker(db, "worker-1", 5*time.Second, publishSvc, storage)

	apiGroup := e.Group("/api/v1")
	humaConfig := huma.DefaultConfig("OpenPost API", "1.0.0")
	api := humaecho.NewWithGroup(e, apiGroup, humaConfig)

	mediaHandler.RegisterRoutes(api)
	mediaHandler.RegisterLegacyRoutes(e)

	e.GET("/openapi.json", func(c echo.Context) error {
		spec := api.OpenAPI()
		data, err := json.Marshal(spec)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to marshal spec"})
		}
		return c.Blob(http.StatusOK, "application/json", data)
	})

	authHandler := handlers.NewAuthHandler(db, authService)
	authHandler.Register(api)
	authHandler.Login(api)
	authHandler.Me(api)

	workspaceHandler := handlers.NewWorkspaceHandler(db, authService)
	workspaceHandler.CreateWorkspace(api)
	workspaceHandler.ListWorkspaces(api)
	workspaceHandler.GetWorkspaceSettings(api)
	workspaceHandler.UpdateWorkspaceSettings(api)

	postHandler := handlers.NewPostHandler(db, authService)
	postHandler.CreatePost(api)
	postHandler.CreateThread(api)
	postHandler.ListPosts(api)
	postHandler.GetPost(api)
	postHandler.UpdatePost(api)
	postHandler.DeletePost(api)
	postHandler.GetScheduleOverview(api)
	postHandler.UpsertVariants(api)
	postHandler.GetVariants(api)
	postHandler.DeleteVariants(api)

	setHandler := handlers.NewSetHandler(db, authService)
	setHandler.CreateSet(api)
	setHandler.ListSets(api)
	setHandler.GetSet(api)
	setHandler.UpdateSet(api)
	setHandler.DeleteSet(api)
	setHandler.AddSetAccounts(api)
	setHandler.RemoveSetAccount(api)

	postingScheduleHandler := handlers.NewPostingScheduleHandler(db, authService)
	postingScheduleHandler.ListSchedules(api)
	postingScheduleHandler.CreateSchedule(api)
	postingScheduleHandler.UpdateSchedule(api)
	postingScheduleHandler.DeleteSchedule(api)
	postingScheduleHandler.SuggestSchedule(api)
	postingScheduleHandler.GetNextAvailableSlot(api)

	promptHandler := handlers.NewPromptHandler(db, authService)
	promptHandler.ListPrompts(api)
	promptHandler.CreatePrompt(api)
	promptHandler.DeletePrompt(api)
	promptHandler.GetRandomPrompt(api)
	promptHandler.GetCategories(api)

	jobHandler := handlers.NewJobHandler(db, authService)
	jobHandler.RegisterRoutes(api)

	oauthHandler := handlers.NewOAuthHandler(db, tokenEncryptor, providers, authService, cfg.DisableLinkedInThreadReplies)
	oauthHandler.ListMastodonServers(api)
	oauthHandler.GetAuthURL(api)
	oauthHandler.Callback(api)
	oauthHandler.ExchangeCode(api)
	oauthHandler.BlueskyLogin(api)
	oauthHandler.ListAccounts(api)
	oauthHandler.DisconnectAccount(api)

	huma.Register(api, huma.Operation{
		OperationID: "health-check",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check",
		Tags:        []string{"System"},
	}, func(_ context.Context, _ *struct{}) (*struct {
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

	RegisterSpaRoutes(e)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Start(ctx)
	}()

	log.Println("Starting OpenPost on :" + cfg.Port)
	log.Println("OpenAPI spec available at http://localhost:" + cfg.Port + "/openapi.json")

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- e.Start(":" + cfg.Port)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case sig := <-sigCh:
		log.Printf("Shutting down after %s...", sig)
	case err := <-serverErrCh:
		if err != nil && err != http.ErrServerClosed {
			cancel()
			signal.Stop(sigCh)
			worker.Stop()
			wg.Wait()
			log.Printf("Server error: %v", err)
			return
		}
		cancel()
		worker.Stop()
		wg.Wait()
		log.Println("Server stopped")
		return
	}

	cancel()
	worker.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Printf("Echo shutdown error: %v", err)
	}

	if err := <-serverErrCh; err != nil && err != http.ErrServerClosed {
		log.Printf("Echo server error: %v", err)
	}

	wg.Wait()
	log.Println("Server stopped")
}
