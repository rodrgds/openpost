package oauth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

type TwitterOAuth struct {
	config *oauth2.Config
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
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

// GenerateAuthURL returns the URL to redirect the user to, and a PKCE code verifier
func (t *TwitterOAuth) GenerateAuthURL(state string) (string, string) {
	// Generate basic standard oauth pkce keys
	codeVerifier := oauth2.GenerateVerifier()
	authURL := t.config.AuthCodeURL(
		state, 
		oauth2.SetAuthURLParam("code_challenge", oauth2.S256ChallengeFromVerifier(codeVerifier)),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return authURL, codeVerifier
}

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}
