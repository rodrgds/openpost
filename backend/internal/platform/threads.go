package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
)

type ThreadsAdapter struct {
	config        *oauth2.Config
	stateStore    sync.Map
	lastUserID    string
	lastUserIDMux sync.Mutex
}

func NewThreadsAdapter(clientID, clientSecret, redirectURI string) *ThreadsAdapter {
	return &ThreadsAdapter{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://www.threads.com/oauth/authorize",
				TokenURL: "https://graph.threads.net/oauth/access_token",
			},
			Scopes: []string{"threads_basic", "threads_content_publish", "threads_manage_replies"},
		},
	}
}

func (t *ThreadsAdapter) GenerateAuthURL(state string) (string, map[string]string) {
	authURL := t.config.AuthCodeURL(state)

	parsedURL, err := url.Parse(authURL)
	if err == nil {
		generatedState := parsedURL.Query().Get("state")
		if generatedState != "" {
			t.stateStore.Store(generatedState, state)
		}
	}

	return authURL, nil
}

func (t *ThreadsAdapter) GetWorkspaceID(state string) (string, bool) {
	value, ok := t.stateStore.Load(state)
	if !ok {
		return "", false
	}
	return value.(string), true
}

func (t *ThreadsAdapter) ExchangeCode(ctx context.Context, code string, extra map[string]string) (*TokenResult, error) {
	values := map[string]string{
		"client_id":     t.config.ClientID,
		"client_secret": t.config.ClientSecret,
		"redirect_uri":  t.config.RedirectURL,
		"code":          code,
		"grant_type":    "authorization_code",
	}

	respBody, err := DoFormURLEncoded(ctx, "POST", t.config.Endpoint.TokenURL, values, nil)
	if err != nil {
		return nil, fmt.Errorf("threads token exchange: %w", err)
	}

	var tokenResp struct {
		AccessToken string      `json:"access_token"`
		UserID      json.Number `json:"user_id"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding threads token: %w", err)
	}

	userID := tokenResp.UserID.String()

	t.lastUserIDMux.Lock()
	t.lastUserID = userID
	t.lastUserIDMux.Unlock()

	longLived, err := t.exchangeLongLivedToken(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("threads long-lived exchange: %w", err)
	}

	return longLived, nil
}

func (t *ThreadsAdapter) exchangeLongLivedToken(ctx context.Context, shortLivedToken string) (*TokenResult, error) {
	params := url.Values{
		"grant_type":    {"th_exchange_token"},
		"client_secret": {t.config.ClientSecret},
		"access_token":  {shortLivedToken},
	}

	respBody, err := DoRequest(ctx, "GET", "https://graph.threads.net/access_token?"+params.Encode(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("threads long-lived token: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding threads long-lived: %w", err)
	}

	return &TokenResult{
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		TokenType:   "Bearer",
		Extra:       map[string]string{"user_id": t.lastUserID},
	}, nil
}

func (t *ThreadsAdapter) RefreshToken(ctx context.Context, accessToken string) (*TokenResult, error) {
	params := url.Values{
		"grant_type":   {"th_refresh_token"},
		"access_token": {accessToken},
	}

	respBody, err := DoRequest(ctx, "GET", "https://graph.threads.net/refresh_access_token?"+params.Encode(), nil, nil)
	if err != nil {
		return nil, fmt.Errorf("threads refresh: %w", err)
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("decoding threads refresh: %w", err)
	}

	return &TokenResult{
		AccessToken: tokenResp.AccessToken,
		ExpiresIn:   tokenResp.ExpiresIn,
		TokenType:   "Bearer",
	}, nil
}

func (t *ThreadsAdapter) GetProfile(ctx context.Context, accessToken string) (*UserProfile, error) {
	endpoint := "https://graph.threads.net/v1.0/me?fields=id,username,name"

	respBody, err := DoRequest(ctx, "GET", endpoint, nil, map[string]string{
		"Authorization": "Bearer " + accessToken,
	})
	if err != nil {
		return nil, err
	}

	var profile struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Name     string `json:"name"`
	}
	if err := json.Unmarshal(respBody, &profile); err != nil {
		return nil, fmt.Errorf("decoding threads profile: %w", err)
	}

	return &UserProfile{
		ID:          profile.ID,
		Username:    profile.Username,
		DisplayName: profile.Name,
	}, nil
}

func (t *ThreadsAdapter) UploadMedia(ctx context.Context, accessToken, accountID, mimeType string, reader io.Reader) (string, error) {
	return "", fmt.Errorf("threads requires publicly accessible URLs, use the media serve URL directly")
}

func (t *ThreadsAdapter) Publish(ctx context.Context, accessToken, userID string, req *PublishRequest) (string, error) {
	isVideo := false
	var mediaURL string

	for _, url := range req.PlatformMediaIDs {
		mediaURL = url
		if isVideoType(url) {
			isVideo = true
		}
	}

	containerID, err := t.createContainer(ctx, accessToken, userID, req.Content, mediaURL, isVideo, req.ReplyToID)
	if err != nil {
		return "", err
	}

	if err := t.waitForContainerReady(ctx, accessToken, containerID); err != nil {
		return "", err
	}

	return t.publishContainer(ctx, accessToken, userID, containerID)
}

func (t *ThreadsAdapter) waitForContainerReady(ctx context.Context, accessToken, containerID string) error {
	const (
		maxAttempts = 10
		pollDelay   = 3 * time.Second
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		statusURL := "https://graph.threads.net/v1.0/" + containerID + "?fields=status,error_message&access_token=" + url.QueryEscape(accessToken)

		respBody, err := DoRequest(ctx, "GET", statusURL, nil, nil)
		if err != nil {
			if attempt == maxAttempts {
				return fmt.Errorf("threads container status check: %w", err)
			}
		} else {
			var statusResp struct {
				Status       string `json:"status"`
				ErrorMessage string `json:"error_message"`
			}
			if err := json.Unmarshal(respBody, &statusResp); err == nil {
				switch statusResp.Status {
				case "FINISHED", "PUBLISHED":
					return nil
				case "ERROR", "EXPIRED", "FAILED":
					if statusResp.ErrorMessage != "" {
						return fmt.Errorf("threads container not publishable: %s", statusResp.ErrorMessage)
					}
					return fmt.Errorf("threads container not publishable: status=%s", statusResp.Status)
				}
			}
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(pollDelay):
			}
		}
	}

	return fmt.Errorf("threads container not ready after %d attempts", maxAttempts)
}

func (t *ThreadsAdapter) createContainer(ctx context.Context, accessToken, userID, content, mediaURL string, isVideo bool, replyToID string) (string, error) {
	payload := map[string]string{
		"text":         content,
		"access_token": accessToken,
	}

	if mediaURL != "" {
		if isVideo {
			payload["media_type"] = "VIDEO"
			payload["video_url"] = mediaURL
		} else {
			payload["media_type"] = "IMAGE"
			payload["image_url"] = mediaURL
		}
	} else {
		payload["media_type"] = "TEXT"
	}

	if replyToID != "" {
		payload["reply_to_id"] = replyToID
	}

	containerURL := "https://graph.threads.net/v1.0/" + userID + "/threads"

	respBody, err := DoFormURLEncoded(ctx, "POST", containerURL, payload, nil)
	if err != nil {
		if replyToID != "" && strings.Contains(err.Error(), `"code":10`) {
			return "", fmt.Errorf("threads container creation (reply permission/check root ownership): %w", err)
		}
		return "", fmt.Errorf("threads container creation: %w", err)
	}

	var containerResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &containerResp); err != nil {
		return "", fmt.Errorf("decoding threads container: %w", err)
	}

	return containerResp.ID, nil
}

func (t *ThreadsAdapter) publishContainer(ctx context.Context, accessToken, userID, creationID string) (string, error) {
	publishURL := "https://graph.threads.net/v1.0/" + userID + "/threads_publish"

	payload := map[string]string{
		"creation_id":  creationID,
		"access_token": accessToken,
	}

	var respBody []byte
	var err error
	const maxPublishAttempts = 5
	for attempt := 1; attempt <= maxPublishAttempts; attempt++ {
		respBody, err = DoFormURLEncoded(ctx, "POST", publishURL, payload, nil)
		if err == nil {
			break
		}

		// Threads may return code 24 briefly right after container creation/status=FINISHED.
		// Retry a few times with short backoff to handle propagation lag.
		if strings.Contains(err.Error(), `"code":24`) && attempt < maxPublishAttempts {
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(time.Duration(attempt) * 2 * time.Second):
			}
			continue
		}

		return "", fmt.Errorf("threads publish: %w", err)
	}

	if err != nil {
		return "", fmt.Errorf("threads publish: %w", err)
	}

	var publishResp struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(respBody, &publishResp); err != nil {
		return "", fmt.Errorf("decoding threads publish: %w", err)
	}

	return publishResp.ID, nil
}

func isVideoType(url string) bool {
	return len(url) > 4 && (url[len(url)-4:] == ".mp4" || url[len(url)-4:] == ".mov" || url[len(url)-5:] == ".webm")
}
