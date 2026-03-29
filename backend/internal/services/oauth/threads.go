package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ThreadsOAuth handles OAuth 2.0 authentication for Meta Threads
type ThreadsOAuth struct {
	config *oauth2.Config
}

// ThreadsProfile represents a Threads user's profile
type ThreadsProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

// ThreadsContainerStatus represents the status of a media container
type ThreadsContainerStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ThreadsPublishingLimit represents rate limit information
type ThreadsPublishingLimit struct {
	Config struct {
		QuotaTotal int `json:"quota_total"`
	} `json:"config"`
	QuotaUsage int `json:"quota_usage"`
}

func NewThreadsOAuth(clientID, clientSecret, redirectURI string) *ThreadsOAuth {
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

// GenerateAuthURL returns the OAuth authorization URL
func (t *ThreadsOAuth) GenerateAuthURL(state string) string {
	return t.config.AuthCodeURL(state)
}

// ExchangeCode exchanges the authorization code for a short-lived token
func (t *ThreadsOAuth) ExchangeCode(ctx context.Context, code string) (*TokenResponse, string, error) {
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
		AccessToken string `json:"access_token"`
		UserID      string `json:"user_id"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, "", err
	}

	return &TokenResponse{
		AccessToken: tokenResp.AccessToken,
		TokenType:   tokenResp.TokenType,
	}, tokenResp.UserID, nil
}

// ExchangeLongLivedToken exchanges a short-lived token for a long-lived token
func (t *ThreadsOAuth) ExchangeLongLivedToken(ctx context.Context, shortLivedToken string) (*TokenResponse, error) {
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

	return &TokenResponse{
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		TokenType:   tokenResp.TokenType,
	}, nil
}

// RefreshToken refreshes a long-lived token
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

// GetProfile fetches the user's Threads profile
func (t *ThreadsOAuth) GetProfile(ctx context.Context, accessToken, userID string) (*ThreadsProfile, error) {
	endpoint := fmt.Sprintf("https://graph.threads.net/v1.0/%s?fields=id,username,name&access_token=%s", userID, accessToken)

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
		return nil, fmt.Errorf("failed to get threads profile: %s", string(body))
	}

	var profile ThreadsProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// CreateMediaContainer creates a media container (Step 1 of Threads posting)
func (t *ThreadsOAuth) CreateMediaContainer(ctx context.Context, accessToken, userID string, mediaType, content, mediaURL string) (string, error) {
	params := url.Values{
		"access_token": {accessToken},
		"media_type":   {mediaType},
	}

	if mediaType == "TEXT" {
		params.Set("text", content)
	} else {
		params.Set("text", content)
		params.Set(mediaType_lower(mediaType)+"_url", mediaURL)
	}

	endpoint := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads?%s", userID, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create media container: %s", string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

// PublishContainer publishes a media container (Step 2 of Threads posting)
func (t *ThreadsOAuth) PublishContainer(ctx context.Context, accessToken, userID, creationID string) (string, error) {
	endpoint := fmt.Sprintf("https://graph.threads.net/v1.0/%s/threads_publish?creation_id=%s&access_token=%s", userID, creationID, accessToken)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to publish container: %s", string(body))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

// PublishTextPost is a convenience method that creates and publishes a text post in one step
func (t *ThreadsOAuth) PublishTextPost(ctx context.Context, accessToken, userID, content string) (string, error) {
	// Create media container
	creationID, err := t.CreateMediaContainer(ctx, accessToken, userID, "TEXT", content, "")
	if err != nil {
		return "", err
	}

	// Publish container
	postID, err := t.PublishContainer(ctx, accessToken, userID, creationID)
	if err != nil {
		return "", err
	}

	return postID, nil
}

// mediaType_lower converts media type to lowercase for URL parameter
func mediaType_lower(mediaType string) string {
	switch mediaType {
	case "IMAGE":
		return "image"
	case "VIDEO":
		return "video"
	default:
		return strings.ToLower(mediaType)
	}
}
