package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type ThreadsProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type ThreadsOAuth struct {
	config     *oauth2.Config
	stateStore sync.Map
}

func NewThreadsOAuth(clientID, clientSecret, redirectURI string) *ThreadsOAuth {
	log.Printf("[ThreadsOAuth] Initializing with client_id=%s, redirect_uri=%s", clientID, redirectURI)
	return &ThreadsOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.threads.com/oauth/authorize",
				TokenURL: "https://graph.threads.net/oauth/access_token",
			},
			Scopes: []string{"threads_basic", "threads_content_publish"},
		},
	}
}

func (t *ThreadsOAuth) GenerateAuthURL(workspaceID string) string {
	authURL := t.config.AuthCodeURL(workspaceID)

	parsedURL, err := url.Parse(authURL)
	if err != nil {
		log.Printf("[ThreadsOAuth] Failed to parse auth URL: %v", err)
		return authURL
	}

	state := parsedURL.Query().Get("state")
	if state != "" {
		t.stateStore.Store(state, workspaceID)
		log.Printf("[ThreadsOAuth] Stored state mapping: state=%s -> workspace_id=%s", state, workspaceID)
	}

	log.Printf("[ThreadsOAuth] Generated auth URL: %s", authURL)
	return authURL
}

func (t *ThreadsOAuth) GetWorkspaceID(state string) (string, bool) {
	value, ok := t.stateStore.Load(state)
	if !ok {
		log.Printf("[ThreadsOAuth] State not found: %s", state)
		return "", false
	}
	log.Printf("[ThreadsOAuth] Found workspace_id for state: %s -> %s", state, value.(string))
	return value.(string), true
}

func (t *ThreadsOAuth) ExchangeCode(ctx context.Context, code string) (*TokenResponse, string, error) {
	log.Printf("[ThreadsOAuth] ExchangeCode called with code length: %d", len(code))

	data := url.Values{
		"client_id":     {t.config.ClientID},
		"client_secret": {t.config.ClientSecret},
		"redirect_uri":  {t.config.RedirectURL},
		"code":          {code},
		"grant_type":    {"authorization_code"},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("threads token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string      `json:"access_token"`
		UserID      interface{} `json:"user_id"`
		TokenType   string      `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, "", err
	}

	var userID string
	switch v := tokenResp.UserID.(type) {
	case float64:
		userID = fmt.Sprintf("%.0f", v)
	case string:
		userID = v
	default:
		return nil, "", fmt.Errorf("unexpected user_id type: %T", tokenResp.UserID)
	}

	log.Printf("[ThreadsOAuth] Exchanged short-lived token for user: %s", userID)

	longLivedToken, err := t.exchangeLongLivedToken(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get long-lived token: %w", err)
	}

	return longLivedToken, userID, nil
}

func (t *ThreadsOAuth) exchangeLongLivedToken(ctx context.Context, shortLivedToken string) (*TokenResponse, error) {
	params := url.Values{
		"grant_type":    {"th_exchange_token"},
		"client_secret": {t.config.ClientSecret},
		"access_token":  {shortLivedToken},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.threads.net/access_token?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to exchange long-lived token: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	log.Printf("[ThreadsOAuth] Got long-lived token, expires in %d seconds", tokenResp.ExpiresIn)

	return &TokenResponse{
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		TokenType:   tokenResp.TokenType,
	}, nil
}

func (t *ThreadsOAuth) RefreshToken(ctx context.Context, accessToken string) (*TokenResponse, error) {
	params := url.Values{
		"grant_type":   {"th_refresh_token"},
		"access_token": {accessToken},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://graph.threads.net/refresh_access_token?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to refresh threads token: %s", string(body))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		TokenType:   tokenResp.TokenType,
	}, nil
}

func (t *ThreadsOAuth) GetProfile(ctx context.Context, accessToken, userID string) (*ThreadsProfile, error) {
	log.Printf("[ThreadsOAuth] GetProfile called for userID: %s", userID)

	// Threads API only allows fetching the authenticated user's profile via /me endpoint
	endpoint := fmt.Sprintf("https://graph.threads.net/v1.0/me?fields=id,username,name&access_token=%s", accessToken)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get profile (%d): %s", resp.StatusCode, string(body))
	}

	var profile ThreadsProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	log.Printf("[ThreadsOAuth] GetProfile succeeded - ID: %s, Username: %s", profile.ID, profile.Username)
	return &profile, nil
}

func (t *ThreadsOAuth) PublishTextPost(ctx context.Context, accessToken, userID, content string) (string, error) {
	log.Printf("[ThreadsOAuth] PublishTextPost called for userID: %s", userID)

	params := url.Values{
		"media_type":   {"TEXT"},
		"text":         {content},
		"access_token": {accessToken},
	}

	containerURL := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads?%s", userID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", containerURL, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("threads container creation failed (%d): %s", resp.StatusCode, string(body))
	}

	var containerResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&containerResp); err != nil {
		return "", err
	}

	log.Printf("[ThreadsOAuth] Created container: %s", containerResp.ID)

	publishURL := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads_publish?creation_id=%s&access_token=%s",
		userID, containerResp.ID, accessToken)

	req, err = http.NewRequestWithContext(ctx, "POST", publishURL, nil)
	if err != nil {
		return "", err
	}

	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("threads publish failed (%d): %s", resp.StatusCode, string(body))
	}

	var publishResp struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&publishResp); err != nil {
		return "", err
	}

	log.Printf("[ThreadsOAuth] Successfully published thread with ID: %s", publishResp.ID)
	return publishResp.ID, nil
}

func (t *ThreadsOAuth) CreateMediaContainer(ctx context.Context, accessToken, userID, mediaType, content, mediaURL string) (string, error) {
	return "", fmt.Errorf("CreateMediaContainer not implemented - use PublishTextPost")
}

func (t *ThreadsOAuth) PublishContainer(ctx context.Context, accessToken, userID, creationID string) (string, error) {
	return "", fmt.Errorf("PublishContainer not implemented - use PublishTextPost")
}
