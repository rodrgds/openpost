package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type TwitterOAuth struct {
	config        *oauth2.Config
	verifierStore sync.Map
}

type TwitterAuthSession struct {
	State        string
	CodeVerifier string
}

type TwitterUser struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
}

type TwitterUserResponse struct {
	Data TwitterUser `json:"data"`
}

func NewTwitterOAuth(clientID, clientSecret, redirectURI string) *TwitterOAuth {
	return &TwitterOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://twitter.com/i/oauth2/authorize",
				TokenURL: "https://api.twitter.com/2/oauth2/token",
			},
			Scopes: []string{"tweet.read", "tweet.write", "users.read", "offline.access"},
		},
	}
}

// GenerateAuthURL returns the URL to redirect the user to, and stores the PKCE verifier
func (t *TwitterOAuth) GenerateAuthURL(state string) (string, string) {
	codeVerifier := oauth2.GenerateVerifier()
	authURL := t.config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return authURL, codeVerifier
}

// StoreVerifier stores the PKCE verifier for a state (call before redirecting user)
func (t *TwitterOAuth) StoreVerifier(state, verifier string) {
	t.verifierStore.Store(state, verifier)
}

// GetVerifier retrieves and removes the PKCE verifier for a state
func (t *TwitterOAuth) GetVerifier(state string) (string, bool) {
	if v, ok := t.verifierStore.Load(state); ok {
		t.verifierStore.Delete(state)
		return v.(string), true
	}
	return "", false
}

// ExchangeCode exchanges the authorization code for tokens
func (t *TwitterOAuth) ExchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {t.config.RedirectURL},
		"code_verifier": {codeVerifier},
		"client_id":     {t.config.ClientID},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.config.ClientID, t.config.ClientSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("twitter token exchange failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}, nil
}

// RefreshToken refreshes an access token
func (t *TwitterOAuth) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {t.config.ClientID},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", t.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(t.config.ClientID, t.config.ClientSecret)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("twitter token refresh failed: %s (status: %d)", string(body), resp.StatusCode)
	}

	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    tokenResp.TokenType,
		Scope:        tokenResp.Scope,
	}, nil
}

// GetMe fetches the authenticated user's profile
func (t *TwitterOAuth) GetMe(ctx context.Context, accessToken string) (*TwitterUser, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.twitter.com/2/users/me?user.fields=id,name,username", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get twitter user: %s", string(body))
	}

	var userResp TwitterUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResp); err != nil {
		return nil, err
	}

	return &userResp.Data, nil
}
