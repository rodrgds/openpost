package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

type MastodonOAuth struct {
	clientID     string
	clientSecret string
	redirectURI  string
}

func NewMastodonOAuth(clientID, clientSecret, redirectURI string) *MastodonOAuth {
	return &MastodonOAuth{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}

// GenerateAuthURL returns auth URL for a specific fediverse instance
func (m *MastodonOAuth) GenerateAuthURL(instanceURL, state string) string {
	config := &oauth2.Config{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		RedirectURL:  m.redirectURI,
		Endpoint: oauth2.Endpoint{
			AuthURL:  instanceURL + "/oauth/authorize",
			TokenURL: instanceURL + "/oauth/token",
		},
		Scopes: []string{"read", "write"},
	}
	return config.AuthCodeURL(state)
}

func (m *MastodonOAuth) ExchangeCode(ctx context.Context, instanceURL, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {m.redirectURI},
		"client_id":     {m.clientID},
		"client_secret": {m.clientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", instanceURL+"/oauth/token", strings.NewReader(data.Encode()))
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
		return nil, fmt.Errorf("failed to exchange token, status code: %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
