package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// BlueskyOAuth handles AT Protocol authentication for Bluesky using app passwords.
type BlueskyOAuth struct {
	pdsURL     string
	httpClient *http.Client
}

// BlueskySession represents an AT Protocol session
type BlueskySession struct {
	Did          string `json:"did"`
	Handle       string `json:"handle"`
	Email        string `json:"email,omitempty"`
	AccessToken  string `json:"accessJwt"`
	RefreshToken string `json:"refreshJwt"`
}

// BlueskyProfile represents a user profile
type BlueskyProfile struct {
	DID    string `json:"did"`
	Handle string `json:"handle"`
}

// BlueskyTokenResponse wraps a session for the token manager
type BlueskyTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func NewBlueskyOAuth(pdsURL string) *BlueskyOAuth {
	if pdsURL == "" {
		pdsURL = "https://bsky.social"
	}
	return &BlueskyOAuth{
		pdsURL:     pdsURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// CreateSession creates a session using an app password
func (b *BlueskyOAuth) CreateSession(ctx context.Context, handle, appPassword string) (*BlueskySession, error) {
	payload := map[string]string{
		"identifier": handle,
		"password":   appPassword,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.server.createSession", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bluesky auth failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var session BlueskySession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// RefreshSession refreshes an expired session
func (b *BlueskyOAuth) RefreshSession(ctx context.Context, refreshToken string) (*BlueskySession, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.server.refreshSession", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+refreshToken)

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bluesky refresh failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var session BlueskySession
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, err
	}

	return &session, nil
}

// PublishPost creates a post on Bluesky
func (b *BlueskyOAuth) PublishPost(ctx context.Context, accessToken, did, content string) (string, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	payload := map[string]interface{}{
		"repo":       did,
		"collection": "app.bsky.feed.post",
		"record": map[string]interface{}{
			"$type":     "app.bsky.feed.post",
			"text":      content,
			"createdAt": now,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", b.pdsURL+"/xrpc/com.atproto.repo.createRecord", strings.NewReader(string(body)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("bluesky post failed (status %d): %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		URI string `json:"uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.URI, nil
}
