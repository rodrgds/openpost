package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/openpost/backend/internal/api/middleware"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/platform"
	account_saver "github.com/openpost/backend/internal/services/account_saver"
	"github.com/openpost/backend/internal/services/auth"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/uptrace/bun"
)

type OAuthHandler struct {
	db                           *bun.DB
	crypto                       *crypto.TokenEncryptor
	providers                    map[string]platform.PlatformAdapter
	auth                         *auth.Service
	disableLinkedInThreadReplies bool
	accountSaver                 *account_saver.AccountSaver
}

func NewOAuthHandler(
	db *bun.DB,
	encryptor *crypto.TokenEncryptor,
	providers map[string]platform.PlatformAdapter,
	authService *auth.Service,
	disableLinkedInThreadReplies bool,
) *OAuthHandler {
	return &OAuthHandler{
		db:                           db,
		crypto:                       encryptor,
		providers:                    providers,
		auth:                         authService,
		disableLinkedInThreadReplies: disableLinkedInThreadReplies,
		accountSaver:                 account_saver.NewAccountSaver(db, encryptor),
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
	OAuthToken string `query:"oauth_token" doc:"OAuth 1.0a request token (X)"`
	Verifier   string `query:"oauth_verifier" doc:"OAuth 1.0a verifier (X)"`
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
	ID                     string `json:"id" doc:"Account ID"`
	Platform               string `json:"platform" doc:"Platform name"`
	AccountID              string `json:"account_id" doc:"Platform-specific account ID"`
	AccountUsername        string `json:"account_username" doc:"Account username"`
	InstanceURL            string `json:"instance_url" doc:"Instance URL (Mastodon/Bluesky)"`
	IsActive               bool   `json:"is_active" doc:"Whether the account is active"`
	ThreadRepliesSupported bool   `json:"thread_replies_supported" doc:"Whether this account supports thread replies in current server config"`
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

		if input.Platform == "x" {
			xAdapter, ok := adapter.(*platform.XAdapter)
			if !ok {
				return nil, huma.Error500InternalServerError("x adapter type mismatch")
			}
			authURL, err := xAdapter.GenerateAuthURLWithError(input.WorkspaceID)
			if err != nil {
				log.Printf("[X OAuth] auth url generation failed: %v", err)
				return nil, huma.Error400BadRequest(fmt.Sprintf("x auth url generation failed: %s", err.Error()))
			}
			resp := &GetAuthURLOutput{}
			resp.Body.URL = authURL
			return resp, nil
		}

		authURL, _ := adapter.GenerateAuthURL(input.WorkspaceID)
		if input.Platform == "mastodon" && input.ServerName != "" {
			authURL, _ = adapter.GenerateAuthURL(input.ServerName + ":" + input.WorkspaceID)
		}
		if authURL == "" {
			return nil, huma.Error400BadRequest(fmt.Sprintf("%s does not support OAuth redirect", input.Platform))
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
		workspaceID := input.State
		if input.Platform == "mastodon" && input.ServerName == "" && input.State != "" {
			parts := strings.SplitN(input.State, ":", 2)
			if len(parts) == 2 {
				input.ServerName = parts[0]
				workspaceID = parts[1]
			}
		}

		adapter, serverName, err := h.getProvider(input.Platform, input.ServerName)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}

		extra := make(map[string]string)
		if input.Platform == "x" {
			extra["oauth_token"] = input.OAuthToken
			extra["oauth_verifier"] = input.Verifier
			if xAdapter, ok := adapter.(*platform.XAdapter); ok {
				workspaceID, ok := xAdapter.GetWorkspaceIDForRequestToken(input.OAuthToken)
				if ok {
					extra["_workspace_id"] = workspaceID
				}
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
	// For Threads, the account ID comes from the token response extra
	if tokenResp.Extra != nil {
		if uid, ok := tokenResp.Extra["user_id"]; ok && uid != "" {
			accountID = uid
		}
	}

	account, err := h.accountSaver.SaveAccount(ctx, platformName, workspaceID, accountID, accountUsername, instanceURL, tokenResp)
	if err != nil {
		log.Printf("[Callback] Failed to save account: %v", err)
		return nil, huma.Error500InternalServerError("failed to save account")
	}

	log.Printf("[Callback] Account saved successfully: ID=%s, redirecting to /", account.ID)

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

		// Delegate saving the account (encrypting tokens and inserting) to AccountSaver
		if _, err := h.accountSaver.SaveAccount(ctx, "mastodon", input.Body.WorkspaceID, profile.ID, profile.Username, input.Body.ServerName, tokenResp); err != nil {
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

		// Build a TokenResult for Bluesky and delegate saving to AccountSaver so encryption and DB insert are centralized
		tokenResp := &platform.TokenResult{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    int(2 * time.Hour / time.Second),
			Extra:        nil,
		}

		if _, err := h.accountSaver.SaveAccount(ctx, "bluesky", input.Body.WorkspaceID, did, input.Body.Handle, "https://bsky.social", tokenResp); err != nil {
			log.Printf("[BlueskyLogin] Failed to save account: %v", err)
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
			threadRepliesSupported := true
			if h.disableLinkedInThreadReplies && acc.Platform == "linkedin" {
				threadRepliesSupported = false
			}

			response[i] = AccountResponse{
				ID:                     acc.ID,
				Platform:               acc.Platform,
				AccountID:              acc.AccountID,
				AccountUsername:        acc.AccountUsername,
				InstanceURL:            acc.InstanceURL,
				IsActive:               acc.IsActive,
				ThreadRepliesSupported: threadRepliesSupported,
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
