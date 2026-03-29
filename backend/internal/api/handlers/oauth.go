package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/openpost/backend/internal/services/oauth"
	"github.com/uptrace/bun"
)

type OAuthHandler struct {
	db              *bun.DB
	crypto          *crypto.TokenEncryptor
	twitter         *oauth.TwitterOAuth
	mastodonServers map[string]*oauth.MastodonOAuth
	bluesky         *oauth.BlueskyOAuth
	linkedin        *oauth.LinkedInOAuth
	threads         *oauth.ThreadsOAuth
	auth            *auth.Service
}

type AuthSessionStore struct {
	sync.Map
}

func NewOAuthHandler(
	db *bun.DB,
	encryptor *crypto.TokenEncryptor,
	tw *oauth.TwitterOAuth,
	mastodonServers map[string]*oauth.MastodonOAuth,
	bs *oauth.BlueskyOAuth,
	li *oauth.LinkedInOAuth,
	th *oauth.ThreadsOAuth,
	authService *auth.Service,
) *OAuthHandler {
	return &OAuthHandler{
		db:              db,
		crypto:          encryptor,
		twitter:         tw,
		mastodonServers: mastodonServers,
		bluesky:         bs,
		linkedin:        li,
		threads:         th,
		auth:            authService,
	}
}

// --- List Mastodon Servers ---

type MastodonServerInfo struct {
	Name        string `json:"name" doc:"Server configuration name"`
	InstanceURL string `json:"instance_url" doc:"Mastodon instance URL"`
}

type ListMastodonServersOutput struct {
	Body []MastodonServerInfo
}

// --- Get Auth URL ---

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

// --- Callback ---

type OAuthCallbackInput struct {
	Platform   string `path:"platform" doc:"Social platform"`
	Code       string `query:"code" doc:"OAuth authorization code"`
	State      string `query:"state" doc:"OAuth state (workspace ID)"`
	ServerName string `query:"server_name" doc:"Mastodon server name (required for mastodon)"`
}

// --- Exchange Code ---

type ExchangeCodeInput struct {
	Body struct {
		WorkspaceID string `json:"workspace_id" doc:"Workspace ID"`
		ServerName  string `json:"server_name" doc:"Mastodon server name from config"`
		Code        string `json:"code" doc:"Authorization code from OAuth flow"`
	}
}

// --- List Accounts ---

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

func (h *OAuthHandler) ListMastodonServers(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-mastodon-servers",
		Method:      http.MethodGet,
		Path:        "/accounts/mastodon/servers",
		Summary:     "List configured Mastodon servers",
		Tags:        []string{"Accounts"},
		Middlewares: huma.Middlewares{middleware.AuthMiddleware(api, h.auth)},
	}, func(ctx context.Context, input *struct{}) (*ListMastodonServersOutput, error) {
		servers := make([]MastodonServerInfo, 0, len(h.mastodonServers))
		for name, provider := range h.mastodonServers {
			servers = append(servers, MastodonServerInfo{
				Name:        name,
				InstanceURL: provider.InstanceURL(),
			})
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
		state := input.WorkspaceID

		switch input.Platform {
		case "x":
			url, verifier := h.twitter.GenerateAuthURL(state)
			h.twitter.StoreVerifier(state, verifier)
			resp := &GetAuthURLOutput{}
			resp.Body.URL = url
			return resp, nil

		case "mastodon":
			provider, ok := h.mastodonServers[input.ServerName]
			if !ok {
				return nil, huma.Error400BadRequest("unknown mastodon server name")
			}
			url := provider.GenerateAuthURL(state)
			resp := &GetAuthURLOutput{}
			resp.Body.URL = url
			return resp, nil

		case "bluesky":
			return nil, huma.Error400BadRequest("bluesky uses app passwords, not OAuth redirect")

		case "linkedin":
			if h.linkedin == nil {
				return nil, huma.Error400BadRequest("linkedin not configured")
			}
			url := h.linkedin.GenerateAuthURL(state)
			resp := &GetAuthURLOutput{}
			resp.Body.URL = url
			return resp, nil

		case "threads":
			if h.threads == nil {
				return nil, huma.Error400BadRequest("threads not configured")
			}
			url := h.threads.GenerateAuthURL(state)
			resp := &GetAuthURLOutput{}
			resp.Body.URL = url
			return resp, nil

		default:
			return nil, huma.Error400BadRequest("unsupported platform")
		}
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
		var tokenResp *oauth.TokenResponse
		var err error
		var instanceURL string
		var accountID string
		var accountUsername string

		switch input.Platform {
		case "x":
			verifier, ok := h.twitter.GetVerifier(input.State)
			if !ok {
				return nil, huma.Error400BadRequest("invalid or expired state")
			}
			tokenResp, err = h.twitter.ExchangeCode(ctx, input.Code, verifier)
			if err != nil {
				return nil, huma.Error500InternalServerError("token exchange failed")
			}
			user, err := h.twitter.GetMe(ctx, tokenResp.AccessToken)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to get user profile")
			}
			accountID = user.ID
			accountUsername = user.Username

		case "mastodon":
			provider, ok := h.mastodonServers[input.ServerName]
			if !ok {
				return nil, huma.Error400BadRequest("unknown mastodon server name")
			}
			instanceURL = provider.InstanceURL()
			tokenResp, err = provider.ExchangeCode(ctx, input.Code)
			if err != nil {
				return nil, huma.Error500InternalServerError(fmt.Sprintf("mastodon token exchange failed: %s", err.Error()))
			}
			profile, err := provider.GetProfile(ctx, tokenResp.AccessToken)
			if err != nil {
				// Non-fatal: profile fetch failed, use defaults
				accountID = "mastodon-user"
				accountUsername = ""
			} else {
				accountID = profile.ID
				accountUsername = profile.Acct
			}

		case "linkedin":
			if h.linkedin == nil {
				return nil, huma.Error400BadRequest("linkedin not configured")
			}
			tokenResp, err = h.linkedin.ExchangeCode(ctx, input.Code)
			if err != nil {
				return nil, huma.Error500InternalServerError("token exchange failed")
			}
			profile, err := h.linkedin.GetProfile(ctx, tokenResp.AccessToken)
			if err != nil {
				return nil, huma.Error500InternalServerError("failed to get profile")
			}
			accountID = profile.ID
			accountUsername = profile.Name

		case "threads":
			if h.threads == nil {
				return nil, huma.Error400BadRequest("threads not configured")
			}
			log.Printf("[Callback] Threads callback - code received, state: %s", input.State)

			workspaceID, ok := h.threads.GetWorkspaceID(input.State)
			if !ok {
				log.Printf("[Callback] Threads state not found in store")
				return nil, huma.Error400BadRequest("invalid or expired state")
			}
			log.Printf("[Callback] Threads retrieved workspace_id: %s", workspaceID)

			var userID string
			tokenResp, userID, err = h.threads.ExchangeCode(ctx, input.Code)
			if err != nil {
				log.Printf("[Callback] Threads ExchangeCode error: %v", err)
				return nil, huma.Error500InternalServerError("token exchange failed")
			}
			log.Printf("[Callback] Threads token exchanged successfully, userID: %s", userID)
			profile, err := h.threads.GetProfile(ctx, tokenResp.AccessToken, userID)
			if err != nil {
				log.Printf("[Callback] Threads GetProfile error: %v", err)
				return nil, huma.Error500InternalServerError("failed to get profile")
			}
			log.Printf("[Callback] Threads profile retrieved: ID=%s, Username=%s", profile.ID, profile.Username)
			accountID = profile.ID
			accountUsername = profile.Username
			input.State = workspaceID

		default:
			return nil, huma.Error400BadRequest("unsupported platform")
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
			expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.State,
			Platform:        input.Platform,
			AccountID:       accountID,
			AccountUsername: accountUsername,
			InstanceURL:     instanceURL,
			AccessTokenEnc:  encAccess,
			RefreshTokenEnc: encRefresh,
			TokenExpiresAt:  expiresAt,
			IsActive:        true,
			CreatedAt:       time.Now(),
		}

		log.Printf("[Callback] Saving %s account: ID=%s, PlatformAccountID=%s, Username=%s",
			input.Platform, account.ID, accountID, accountUsername)

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
	})
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
		// This endpoint only handles Mastodon exchange
		// Other platforms use the OAuth redirect flow
		provider, ok := h.mastodonServers[input.Body.ServerName]
		if !ok {
			return nil, huma.Error400BadRequest("unknown mastodon server name")
		}

		instanceURL := provider.InstanceURL()
		tokenResp, err := provider.ExchangeCode(ctx, input.Body.Code)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("mastodon exchange failed: %s", err.Error()))
		}

		var accountID string
		var accountUsername string
		profile, err := provider.GetProfile(ctx, tokenResp.AccessToken)
		if err != nil {
			// Non-fatal: profile fetch failed, use defaults
			accountID = "mastodon-user"
			accountUsername = ""
		} else {
			accountID = profile.ID
			accountUsername = profile.Acct
		}

		if err != nil {
			return nil, huma.Error500InternalServerError("token exchange failed")
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
			expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.Body.WorkspaceID,
			Platform:        "mastodon",
			AccountID:       accountID,
			AccountUsername: accountUsername,
			InstanceURL:     instanceURL,
			AccessTokenEnc:  encAccess,
			RefreshTokenEnc: encRefresh,
			TokenExpiresAt:  expiresAt,
			IsActive:        true,
			CreatedAt:       time.Now(),
		}

		if _, err := h.db.NewInsert().Model(account).Exec(ctx); err != nil {
			return nil, huma.Error500InternalServerError("failed to save account")
		}

		return nil, nil
	})
}

// --- Bluesky Login (app password) ---

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
		if h.bluesky == nil {
			return nil, huma.Error400BadRequest("bluesky not configured")
		}

		session, err := h.bluesky.CreateSession(ctx, input.Body.Handle, input.Body.AppPassword)
		if err != nil {
			return nil, huma.Error500InternalServerError(fmt.Sprintf("bluesky login failed: %s", err.Error()))
		}

		encAccess, err := h.crypto.Encrypt(session.AccessToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		encRefresh, err := h.crypto.Encrypt(session.RefreshToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.Body.WorkspaceID,
			Platform:        "bluesky",
			AccountID:       session.Did,
			AccountUsername: session.Handle,
			InstanceURL:     "https://bsky.social",
			AccessTokenEnc:  encAccess,
			RefreshTokenEnc: encRefresh,
			IsActive:        true,
			CreatedAt:       time.Now(),
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

// --- Disconnect Account ---

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
