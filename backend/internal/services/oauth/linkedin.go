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

// LinkedInOAuth handles OAuth 2.0 authentication for LinkedIn
type LinkedInOAuth struct {
	config *oauth2.Config
}

// LinkedInProfile represents a LinkedIn user's basic profile
type LinkedInProfile struct {
	ID   string `json:"id"`
	Name string `json:"localizedFirstName"`
}

func NewLinkedInOAuth(clientID, clientSecret, redirectURI string) *LinkedInOAuth {
	return &LinkedInOAuth{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.linkedin.com/oauth/v2/authorization",
				TokenURL: "https://www.linkedin.com/oauth/v2/accessToken",
			},
			Scopes: []string{"openid", "profile", "w_member_social"},
		},
	}
}

// GenerateAuthURL returns the OAuth authorization URL
func (l *LinkedInOAuth) GenerateAuthURL(state string) string {
	return l.config.AuthCodeURL(state)
}

// ExchangeCode exchanges the authorization code for tokens
func (l *LinkedInOAuth) ExchangeCode(ctx context.Context, code string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {l.config.RedirectURL},
		"client_id":     {l.config.ClientID},
		"client_secret": {l.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("linkedin token exchange failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken           string `json:"access_token"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
		Scope                 string `json:"scope"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    "Bearer",
		Scope:        tokenResp.Scope,
	}, nil
}

// RefreshToken refreshes an access token
func (l *LinkedInOAuth) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {l.config.ClientID},
		"client_secret": {l.config.ClientSecret},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", l.config.Endpoint.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("linkedin token refresh failed: %s", string(body))
	}

	var tokenResp struct {
		AccessToken           string `json:"access_token"`
		ExpiresIn             int    `json:"expires_in"`
		RefreshToken          string `json:"refresh_token"`
		RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &TokenResponse{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		TokenType:    "Bearer",
	}, nil
}

// GetProfile fetches the user's LinkedIn profile
func (l *LinkedInOAuth) GetProfile(ctx context.Context, accessToken string) (*LinkedInProfile, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.linkedin.com/v2/userinfo", nil)
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
		return nil, fmt.Errorf("failed to get linkedin profile: %s", string(body))
	}

	var profile struct {
		Sub           string `json:"sub"`
		Name          string `json:"name"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Picture       string `json:"picture"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return nil, err
	}

	return &LinkedInProfile{
		ID:   profile.Sub,
		Name: profile.Name,
	}, nil
}

// PublishPost creates a post on LinkedIn using the Posts API
func (l *LinkedInOAuth) PublishPost(ctx context.Context, accessToken, personID, content string) (string, error) {
	payload := map[string]interface{}{
		"author":     fmt.Sprintf("urn:li:person:%s", personID),
		"commentary": content,
		"visibility": "PUBLIC",
		"distribution": map[string]interface{}{
			"feedDistribution":               "MAIN_FEED",
			"targetEntities":                 []interface{}{},
			"thirdPartyDistributionChannels": []interface{}{},
		},
		"lifecycleState":            "PUBLISHED",
		"isReshareDisabledByAuthor": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.linkedin.com/rest/posts", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Restli-Protocol-Version", "2.0.0")
	req.Header.Set("Linkedin-Version", "202401")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create linkedin post: %s", string(respBody))
	}

	// Post ID is in the x-restli-id header
	postID := resp.Header.Get("x-restli-id")
	return postID, nil
}
