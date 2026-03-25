package handlers

import (
	"context"
	"net/http"
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
	db       *bun.DB
	crypto   *crypto.TokenEncryptor
	twitter  *oauth.TwitterOAuth
	mastodon *oauth.MastodonOAuth
	auth     *auth.Service
}

func NewOAuthHandler(db *bun.DB, encryptor *crypto.TokenEncryptor, tw *oauth.TwitterOAuth, ma *oauth.MastodonOAuth, authService *auth.Service) *OAuthHandler {
	return &OAuthHandler{
		db:       db,
		crypto:   encryptor,
		twitter:  tw,
		mastodon: ma,
		auth:     authService,
	}
}

// --- Get Auth URL ---

type GetAuthURLInput struct {
	Platform    string `path:"platform" doc:"Social platform (x, mastodon)"`
	WorkspaceID string `query:"workspace_id" doc:"Workspace ID to link account to"`
	Instance    string `query:"instance" doc:"Mastodon instance URL (required for mastodon)"`
}

type GetAuthURLOutput struct {
	Body struct {
		URL string `json:"url" doc:"OAuth authorization URL"`
	}
}

// --- Callback ---

type OAuthCallbackInput struct {
	Platform string `path:"platform" doc:"Social platform"`
	Code     string `query:"code" doc:"OAuth authorization code"`
	State    string `query:"state" doc:"OAuth state (workspace ID)"`
}

// --- Exchange Code ---

type ExchangeCodeInput struct {
	Body struct {
		WorkspaceID string `json:"workspace_id" doc:"Workspace ID"`
		Instance    string `json:"instance" doc:"Mastodon instance URL"`
		Code        string `json:"code" doc:"Authorization code from OOB flow"`
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
			resp := &GetAuthURLOutput{}
			resp.Body.URL = url
			echoCtx := ctx.Value("echo_context")
			if echoCtx != nil {
				// Not available in pure Huma context, cookie setting handled by adapter
			}
			_ = verifier // verifier cookie would be set via echo.Context
			return resp, nil

		case "mastodon":
			if input.Instance == "" {
				return nil, huma.Error400BadRequest("instance query param required for mastodon")
			}
			url := h.mastodon.GenerateAuthURL(input.Instance, state)
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
		Hidden:      true, // Called by OAuth providers, not by frontend
	}, func(ctx context.Context, input *OAuthCallbackInput) (*huma.StreamResponse, error) {
		var tokenResp *oauth.TokenResponse
		var err error
		var instanceURL string

		switch input.Platform {
		case "x":
			tokenResp, err = h.twitter.ExchangeCode(ctx, input.Code, "")
		case "mastodon":
			tokenResp, err = h.mastodon.ExchangeCode(ctx, instanceURL, input.Code)
		default:
			return nil, huma.Error400BadRequest("unsupported platform")
		}

		if err != nil {
			return nil, huma.Error500InternalServerError("token exchange failed")
		}

		encAccess, err := h.crypto.Encrypt(tokenResp.AccessToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		encRefresh, err := h.crypto.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		var expiresAt time.Time
		if tokenResp.ExpiresIn > 0 {
			expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.State,
			Platform:        input.Platform,
			AccountID:       "fetch-id-via-profile-api",
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

		// Redirect to frontend
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
		tokenResp, err := h.mastodon.ExchangeCode(ctx, input.Body.Instance, input.Body.Code)
		if err != nil {
			return nil, huma.Error500InternalServerError("token exchange failed")
		}

		encAccess, err := h.crypto.Encrypt(tokenResp.AccessToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		encRefresh, err := h.crypto.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			return nil, huma.Error500InternalServerError("encryption failed")
		}

		var expiresAt time.Time
		if tokenResp.ExpiresIn > 0 {
			expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		}

		account := &models.SocialAccount{
			ID:              uuid.New().String(),
			WorkspaceID:     input.Body.WorkspaceID,
			Platform:        "mastodon",
			AccountID:       "fetch-id-via-profile-api",
			InstanceURL:     input.Body.Instance,
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
