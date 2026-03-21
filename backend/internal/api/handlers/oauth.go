package handlers

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/openpost/backend/internal/services/oauth"
	"github.com/uptrace/bun"
)

type OAuthHandler struct {
	db       *bun.DB
	crypto   *crypto.TokenEncryptor
	twitter  *oauth.TwitterOAuth
	mastodon *oauth.MastodonOAuth
}

func NewOAuthHandler(db *bun.DB, encryptor *crypto.TokenEncryptor, tw *oauth.TwitterOAuth, ma *oauth.MastodonOAuth) *OAuthHandler {
	return &OAuthHandler{
		db:       db,
		crypto:   encryptor,
		twitter:  tw,
		mastodon: ma,
	}
}

// StartAuth generates the OAuth URL and redirects the user (deprecated - use GetAuthURL)
func (h *OAuthHandler) StartAuth(c echo.Context) error {
	platform := c.Param("platform")
	workspaceID := c.QueryParam("workspace_id")

	if workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id is required"})
	}

	// Pass workspaceID in state to remember where to link the account
	state := workspaceID

	switch platform {
	case "x":
		url, verifier := h.twitter.GenerateAuthURL(state)
		// We store the verifier in a cookie for the callback to read
		c.SetCookie(&http.Cookie{Name: "oauth_verifier", Value: verifier, Path: "/api/v1/accounts/x/callback", HttpOnly: true, MaxAge: 300})
		return c.Redirect(http.StatusTemporaryRedirect, url)

	case "mastodon":
		instance := c.QueryParam("instance")
		if instance == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance query param required for mastodon"})
		}

		url := h.mastodon.GenerateAuthURL(instance, state)
		c.SetCookie(&http.Cookie{Name: "mastodon_instance", Value: instance, Path: "/api/v1/accounts/mastodon/callback", HttpOnly: true, MaxAge: 300})
		return c.Redirect(http.StatusTemporaryRedirect, url)

	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unsupported platform"})
	}
}

// GetAuthURL returns the OAuth URL as JSON (for API calls with auth header)
func (h *OAuthHandler) GetAuthURL(c echo.Context) error {
	platform := c.Param("platform")
	workspaceID := c.QueryParam("workspace_id")

	if workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id is required"})
	}

	// State contains workspace ID for callback
	state := workspaceID

	switch platform {
	case "x":
		url, verifier := h.twitter.GenerateAuthURL(state)
		c.SetCookie(&http.Cookie{Name: "oauth_verifier", Value: verifier, Path: "/api/v1/accounts/x/callback", HttpOnly: true, MaxAge: 300})
		return c.JSON(http.StatusOK, map[string]string{"url": url})

	case "mastodon":
		instance := c.QueryParam("instance")
		if instance == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "instance query param required for mastodon"})
		}

		url := h.mastodon.GenerateAuthURL(instance, state)
		c.SetCookie(&http.Cookie{Name: "mastodon_instance", Value: instance, Path: "/api/v1/accounts/mastodon/callback", HttpOnly: true, MaxAge: 300})
		return c.JSON(http.StatusOK, map[string]string{"url": url})

	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unsupported platform"})
	}
}

// ExchangeCode handles manual code exchange for OOB flow
func (h *OAuthHandler) ExchangeCode(c echo.Context) error {
	var req struct {
		WorkspaceID string `json:"workspace_id"`
		Instance    string `json:"instance"`
		Code        string `json:"code"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request"})
	}

	if req.WorkspaceID == "" || req.Instance == "" || req.Code == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id, instance, and code are required"})
	}

	ctx := c.Request().Context()

	tokenResp, err := h.mastodon.ExchangeCode(ctx, req.Instance, req.Code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "token exchange failed: " + err.Error()})
	}

	encAccess, err := h.crypto.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
	}

	encRefresh, err := h.crypto.Encrypt(tokenResp.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	account := &models.SocialAccount{
		ID:              uuid.New().String(),
		WorkspaceID:     req.WorkspaceID,
		Platform:        "mastodon",
		AccountID:       "fetch-id-via-profile-api",
		InstanceURL:     req.Instance,
		AccessTokenEnc:  encAccess,
		RefreshTokenEnc: encRefresh,
		TokenExpiresAt:  expiresAt,
		IsActive:        true,
		CreatedAt:       time.Now(),
	}

	if _, err := h.db.NewInsert().Model(account).Exec(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db insert failed"})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// Callback handles the OAuth redirect back from the provider
func (h *OAuthHandler) Callback(c echo.Context) error {
	platform := c.Param("platform")
	code := c.QueryParam("code")
	workspaceID := c.QueryParam("state")

	if code == "" || workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing code or state"})
	}

	ctx := c.Request().Context()
	var tokenResp *oauth.TokenResponse
	var err error
	var instanceURL string

	switch platform {
	case "x":
		verifierCookie, errReader := c.Cookie("oauth_verifier")
		if errReader != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing verifier cookie"})
		}

		tokenResp, err = h.twitter.ExchangeCode(ctx, code, verifierCookie.Value)

	case "mastodon":
		instanceCookie, errReader := c.Cookie("mastodon_instance")
		if errReader != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing mastodon instance cookie"})
		}
		instanceURL = instanceCookie.Value
		tokenResp, err = h.mastodon.ExchangeCode(ctx, instanceURL, code)

	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Unsupported platform"})
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "token exchange failed: " + err.Error()})
	}

	// Encrypt tokens
	encAccess, err := h.crypto.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
	}

	encRefresh, err := h.crypto.Encrypt(tokenResp.RefreshToken)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "encryption failed"})
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	account := &models.SocialAccount{
		ID:              uuid.New().String(),
		WorkspaceID:     workspaceID,
		Platform:        platform,
		AccountID:       "fetch-id-via-profile-api", // Real app fetches /me profile here
		InstanceURL:     instanceURL,
		AccessTokenEnc:  encAccess,
		RefreshTokenEnc: encRefresh,
		TokenExpiresAt:  expiresAt,
		IsActive:        true,
		CreatedAt:       time.Now(),
	}

	if _, err := h.db.NewInsert().Model(account).Exec(ctx); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "db insert failed"})
	}

	// Redirect back to frontend dashboard
	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

// ListAccounts returns all connected social accounts for a workspace
func (h *OAuthHandler) ListAccounts(c echo.Context) error {
	workspaceID := c.QueryParam("workspace_id")
	if workspaceID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "workspace_id is required"})
	}

	ctx := c.Request().Context()
	var accounts []models.SocialAccount

	err := h.db.NewSelect().
		Model(&accounts).
		Where("workspace_id = ?", workspaceID).
		Where("is_active = ?", true).
		Order("created_at DESC").
		Scan(ctx)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to list accounts"})
	}

	type AccountResponse struct {
		ID              string `json:"id"`
		Platform        string `json:"platform"`
		AccountID       string `json:"account_id"`
		AccountUsername string `json:"account_username"`
		InstanceURL     string `json:"instance_url"`
		IsActive        bool   `json:"is_active"`
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

	return c.JSON(http.StatusOK, response)
}
