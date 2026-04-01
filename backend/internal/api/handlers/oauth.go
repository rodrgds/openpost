package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/platform"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/uptrace/bun"
)

type OAuthHandler struct {
	db        *bun.DB
	crypto    *crypto.TokenEncryptor
	providers map[string]platform.PlatformAdapter
	auth      *auth.Service
}

func NewOAuthHandler(
	db *bun.DB,
	encryptor *crypto.TokenEncryptor,
	providers map[string]platform.PlatformAdapter,
	authService *auth.Service,
) *OAuthHandler {
	return &OAuthHandler{
		db:        db,
		crypto:    encryptor,
		providers: providers,
		auth:      authService,
	}
}

type MastodonServerInfo struct {
	Name        string `json:"name" doc:"Server configuration name"`
	InstanceURL string `json:"instance_url" doc:"Mastodon instance URL"`
}

type ListMastodonServersOutput struct {
	Body []MastodonServerInfo
}

type GetAuthURLInput struct {
	Platform    string `path:"platform" doc:"Social platform (x, mastodon, bluesky, linkedin, threads)"`
	WorkspaceID string `query:"workspace_id" doc:"Workspace ID to link account to"`
	ServerName  string `query:"server_name" doc:"Mastodon server name from config (required for mastodon)"`
}

type GetAuthURLOutput struct {
	Body struct {
		URL string `json:"url" doc:"OAuth authorization URL"`
	}
}

type OAuthCallbackInput struct {
	Platform   string `path:"platform" doc:"Social platform"`
	Code       string `query:"code" doc:"OAuth authorization code"`
	State      string `query:"state" doc:"OAuth state (workspace ID)"`
	ServerName string `query:"server_name" doc:"Mastodon server name (required for mastodon)"`
}

type ExchangeCodeInput struct {
	Body struct {
		WorkspaceID string `json:"workspace_id" doc:"Workspace ID"`
		ServerName  string `json:"server_name" doc:"Mastodon server name from config"`
		Code        string `json:"code" doc:"Authorization code from OAuth flow"`
	}
}

type ListAccountsInput struct {
	WorkspaceID string `query:"workspace_id" doc:"Filter by workspace ID"`
}

type AccountResponse struct {
	ID              string `json:"id" doc:"Account ID"`
	Platform        string `json:"platform" doc:"Platform name"`
	AccountID       string `json:"account_id" doc:"Platform-specific account ID"`
	AccountUsername string `json:"account_username" doc:"Account username"`
	InstanceURL     string `json:"instance_url" doc:"Instance URL (Mastodon/Bluesky)"`
	IsActive        bool   `json:"is_active" doc:"Whether the account is active"`
}

type ListAccountsOutput struct {
	Body []AccountResponse
}

func (h *OAuthHandler) getProvider(platform, serverName string) (platform.PlatformAdapter, string, error) {
	if platform == "mastodon" {
		if serverName == "" {
			return nil, "", fmt.Errorf("server_name required for mastodon")
		}
		key := "mastodon:" + serverName
		adapter, ok := h.providers[key]
		if !ok {
			return nil, "", fmt.Errorf("unknown mastodon server: %s", serverName)
		}
		return adapter, serverName, nil
	}

	adapter, ok := h.providers[platform]
	if !ok {
		return nil, "", fmt.Errorf("unsupported platform: %s", platform)
	}
	return adapter, "", nil
}

func (h *OAuthHandler) ListMastodonServers(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-mastodon-servers",
		Method:      http.MethodGet,
		Path:        "/accounts/mastodon/servers",
		Summary:     "List configured Mastodon servers",
		Tags:        []string{"Accounts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *struct{}) (*ListMastodonServersOutput, error) {
		var servers []MastodonServerInfo
		for key, adapter := range h.providers {
			if !strings.HasPrefix(key, "mastodon:") {
				continue
			}
			if m, ok := adapter.(interface{ InstanceURL() string }); ok {
				name := strings.TrimPrefix(key, "mastodon:")
				servers = append(servers, MastodonServerInfo{
					Name:        name,
					InstanceURL: m.InstanceURL(),
				})
			}
		}
		return &ListMastodonServersOutput{Body: servers}, nil
	})
}

func (h *OAuthHandler) GetAuthURL(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-auth-url",
		Method:      http.MethodGet,
		Path:        "/accounts/{platform}/auth-url",
		Summary:     "Get OAuth authorization URL for a platform",
		Tags:        []string{"Accounts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{400},
	}, func(ctx context.Context, input *GetAuthURLInput) (*GetAuthURLOutput, error) {
		if input.Platform == "bluesky" {
			return nil, huma.Error400BadRequest("bluesky uses app passwords, not OAuth redirect")
		}

		adapter, _, err := h.getProvider(input.Platform, input.ServerName)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		authURL, extra := adapter.GenerateAuthURL(input.WorkspaceID)
		if authURL == "" {
			return nil, huma.Error400BadRequest(fmt.Sprintf("%s does not support OAuth redirect", input.Platform))
		}

		if input.Platform == "x" {
			if verifier, ok := extra["code_verifier"]; ok {
				if xAdapter, ok := adapter.(*platform.XAdapter); ok {
					xAdapter.StoreVerifier(input.WorkspaceID, verifier)
				}
			}
		}

		resp := &GetAuthURLOutput{}
		resp.Body.URL = authURL
		return resp, nil
	})
}

func (h *OAuthHandler) Callback(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "oauth-callback",
		Method:      http.MethodGet,
		Path:        "/accounts/{platform}/callback",
		Summary:     "Handle OAuth callback from provider",
		Tags:        []string{"Accounts"},
		Errors:      []int{400},
		Hidden:      true,
	}, func(ctx context.Context, input *OAuthCallbackInput) (*huma.StreamResponse, error) {
		adapter, serverName, err := h.getProvider(input.Platform, input.ServerName)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		extra := make(map[string]string)
		if input.Platform == "x" {
			if xAdapter, ok := adapter.(*platform.XAdapter); ok {
				verifier, ok := xAdapter.GetVerifier(input.State)
				if !ok {
					return nil, huma.Error400BadRequest("invalid or expired state")
				}
				extra["code_verifier"] = verifier
			}
		}

		if input.Platform == "threads" {
			if threadsAdapter, ok := adapter.(*platform.ThreadsAdapter); ok {
				workspaceID, ok := threadsAdapter.GetWorkspaceID(input.State)
				if !ok {
					return nil, huma.Error400BadRequest("invalid or expired state")
				}
				extra["_workspace_id"] = workspaceID
			}
		}

		tokenResp, err := adapter.ExchangeCode(ctx, input.Code, extra)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("token exchange failed: %s", err.Error()))
		}

		profile, err := adapter.GetProfile(ctx, tokenResp.AccessToken)
		if err != nil {
			if input.Platform == "mastodon" {
				profile = &platform.UserProfile{ID: "mastodon-user", Username: ""}
			} else {
				return nil, huma.Error500InternalServerError(fmt.Sprintf("failed to get profile: %s", err.Error()))
			}
		}

		workspaceID := input.State
		if ws, ok := extra["_workspace_id"]; ok {
			workspaceID = ws
		}

		instanceRef := ""
		if serverName != "" {
			instanceRef = serverName
		}

		return h.saveAccountAndRedirect(ctx, input.Platform, workspaceID, profile.ID, profile.Username, instanceRef, tokenResp)
	})
}

func (h *OAuthHandler) saveAccountAndRedirect(ctx context.Context, platformName, workspaceID, accountID, accountUsername, instanceURL string, tokenResp *platform.TokenResult) (*huma.StreamResponse, error) {
	encAccess, err := h.crypto.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return nil, huma.Error500InternalServerError("encryption failed")
	}

	var encRefresh []byte
	if tokenResp.RefreshToken != "" {
		encRefresh, err = h.crypto.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	account := &models.SocialAccount{
		ID:              uuid.New().String(),
		WorkspaceID:     workspaceID,
		Platform:        platformName,
		AccountID:       accountID,
		AccountUsername: accountUsername,
		InstanceURL:     instanceURL,
		AccessTokenEnc:  encAccess,
		RefreshTokenEnc: encRefresh,
		TokenExpiresAt:  expiresAt,
		IsActive:        true,
		CreatedAt:       time.Now().UTC(),
	}

	log.Printf("[Callback] Saving %s account: ID=%s, PlatformAccountID=%s, Username=%s",
		platformName, account.ID, accountID, accountUsername)

	if _, err := h.db.NewInsert().Model(account).Exec(ctx); err != nil {
		log.Printf("[Callback] Failed to save account: %v", err)
		return nil, huma.Error500InternalServerError("failed to save account")
	}

	log.Printf("[Callback] Account saved successfully, redirecting to /")

	return &huma.StreamResponse{
		Body: func(ctx huma.Context) {
			ctx.SetStatus(http.StatusTemporaryRedirect)
			ctx.SetHeader("Location", "/")
		},
	}, nil
}

func (h *OAuthHandler) ExchangeCode(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "exchange-mastodon-code",
		Method:      http.MethodPost,
		Path:        "/accounts/mastodon/exchange",
		Summary:     "Exchange Mastodon OOB authorization code",
		Tags:        []string{"Accounts"},
		Errors:      []int{400},
	}, func(ctx context.Context, input *ExchangeCodeInput) (*struct{}, error) {
		adapter, _, err := h.getProvider("mastodon", input.Body.ServerName)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		tokenResp, err := adapter.ExchangeCode(ctx, input.Body.Code, nil)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("mastodon exchange failed: %s", err.Error()))
		}

		profile, err := adapter.GetProfile(ctx, tokenResp.AccessToken)
		if err != nil {
			profile = &platform.UserProfile{ID: "mastodon-user", Username: ""}
		}

		encAccess, err := h.crypto.Encrypt(tokenResp.AccessToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		var encRefresh []byte
		if tokenResp.RefreshToken != "" {
			encRefresh, err = h.crypto.Encrypt(tokenResp.RefreshToken)
			if err != nil {
				return nil, huma.Error500InternalServerError("encryption failed")
			}
		}

		var expiresAt time.Time
		if tokenResp.ExpiresIn > 0 {
			expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.Body.WorkspaceID,
			Platform:        "mastodon",
			AccountID:       profile.ID,
			AccountUsername: profile.Username,
			InstanceURL:     input.Body.ServerName,
			AccessTokenEnc:  encAccess,
			RefreshTokenEnc: encRefresh,
			TokenExpiresAt:  expiresAt,
			IsActive:        true,
			CreatedAt:       time.Now().UTC(),
		}

		log.Printf("[ExchangeCode] Saving mastodon account: ID=%s, WorkspaceID=%s, PlatformAccountID=%s, ServerName=%s",
			account.ID, account.WorkspaceID, account.AccountID, input.Body.ServerName)

		if _, err := h.db.NewInsert().Model(account).Exec(ctx); err != nil {
			log.Printf("[ExchangeCode] Failed to save account: %v", err)
			return nil, huma.Error500InternalServerError(fmt.Sprintf("failed to save account: %s", err.Error()))
		}

		log.Printf("[ExchangeCode] Account saved successfully")

		return nil, nil
	})
}

type BlueskyLoginInput struct {
	Body struct {
		WorkspaceID string `json:"workspace_id" doc:"Workspace ID"`
		Handle      string `json:"handle" doc:"Bluesky handle (e.g. user.bsky.social)"`
		AppPassword string `json:"app_password" doc:"Bluesky app password (Settings > App Passwords)"`
	}
}

func (h *OAuthHandler) BlueskyLogin(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "bluesky-login",
		Method:      http.MethodPost,
		Path:        "/accounts/bluesky/login",
		Summary:     "Connect Bluesky account using app password",
		Tags:        []string{"Accounts"},
		Errors:      []int{400},
	}, func(ctx context.Context, input *BlueskyLoginInput) (*struct{}, error) {
		adapter, ok := h.providers["bluesky"]
		if !ok {
			return nil, huma.Error400BadRequest("bluesky not configured")
		}

		blueskyAdapter, ok := adapter.(*platform.BlueskyAdapter)
		if !ok {
			return nil, huma.Error500InternalServerError("bluesky adapter type mismatch")
		}

		did, accessToken, refreshToken, err := blueskyAdapter.CreateSession(ctx, input.Body.Handle, input.Body.AppPassword)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("bluesky login failed: %s", err.Error()))
		}

		encAccess, err := h.crypto.Encrypt(accessToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		encRefresh, err := h.crypto.Encrypt(refreshToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.Body.WorkspaceID,
			Platform:        "bluesky",
			AccountID:       did,
			AccountUsername: input.Body.Handle,
			InstanceURL:     "https://bsky.social",
			AccessTokenEnc:  encAccess,
			RefreshTokenEnc: encRefresh,
			TokenExpiresAt:  time.Now().UTC().Add(2 * time.Hour),
			IsActive:        true,
			CreatedAt:       time.Now().UTC(),
		}

		if _, err := h.db.NewInsert().Model(account).Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to save account")
		}

		return nil, nil
	})
}

func (h *OAuthHandler) ListAccounts(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-accounts",
		Method:      http.MethodGet,
		Path:        "/accounts",
		Summary:     "List connected social accounts for a workspace",
		Tags:        []string{"Accounts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *ListAccountsInput) (*ListAccountsOutput, error) {
		var accounts []models.SocialAccount
		err := h.db.NewSelect().
			Model(&accounts).
			Where("workspace_id = ?", input.WorkspaceID).
			Where("is_active = ?", true).
			Order("created_at DESC").
			Scan(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to list accounts")
		}

		response := make([]AccountResponse, len(accounts))
		for i, acc := range accounts {
			response[i] = AccountResponse{
				ID:              acc.ID,
				Platform:        acc.Platform,
				AccountID:       acc.AccountID,
				AccountUsername: acc.AccountUsername,
				InstanceURL:     acc.InstanceURL,
				IsActive:        acc.IsActive,
			}
		}

		return &ListAccountsOutput{Body: response}, nil
	})
}

type DisconnectAccountInput struct {
	AccountID string `path:"account_id"`
}

func (h *OAuthHandler) DisconnectAccount(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "disconnect-account",
		Method:      http.MethodDelete,
		Path:        "/accounts/{account_id}",
		Summary:     "Disconnect a social account",
		Tags:        []string{"Accounts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
		Errors:      []int{404},
	}, func(ctx context.Context, input *DisconnectAccountInput) (*struct{}, error) {
		result, err := h.db.NewUpdate().
			Model((*models.SocialAccount)(nil)).
			Set("is_active = ?", false).
			Where("id = ?", input.AccountID).
			Exec(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("failed to disconnect account")
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			return nil, huma.Error404NotFound("account not found")
		}

		return nil, nil
	})
}
