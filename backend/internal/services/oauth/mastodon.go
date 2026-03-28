package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

type MastodonOAuth struct {
	clientID     string
	clientSecret string
	redirectURI  string
	instanceURL  string
}

// MastodonProfile represents a user's profile from their instance
type MastodonProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Acct     string `json:"acct"`
}

func NewMastodonOAuth(clientID, clientSecret, redirectURI, instanceURL string) *MastodonOAuth {
	return &MastodonOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
		instanceURL:  instanceURL,
	}
}

// InstanceURL returns the configured instance URL for this provider.
func (m *MastodonOAuth) InstanceURL() string {
	return m.instanceURL
}

// GenerateAuthURL returns the OAuth authorization URL for this server.
func (m *MastodonOAuth) GenerateAuthURL(state string) string {
	config := &oauth2.Config{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		RedirectURL:  m.redirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  m.instanceURL + "/oauth/authorize",
			TokenURL: m.instanceURL + "/oauth/token",
		},
		Scopes: []string{"read", "write"},
	}
	return config.AuthCodeURL(state)
}

func (m *MastodonOAuth) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {m.redirectURI},
		"client_id":     {m.clientID},
		"client_secret": {m.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.instanceURL+"/oauth/token", strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mastodon token exchange failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	return &tokenResp, nil
}

// GetProfile fetches the authenticated user's profile
func (m *MastodonOAuth) GetProfile(ctx context.Context, accessToken string) (*MastodonProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", m.instanceURL+"/api/v1/accounts/verify_credentials", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get mastodon profile (status %d): %s", resp.StatusCode, string(body))
	}

	var profile MastodonProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return &profile, nil
}
