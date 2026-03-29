package oauth

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	threads "github.com/tirthpatell/threads-go"
)

type ThreadsProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type ThreadsOAuth struct {
	config     *threads.Config
	stateStore sync.Map
}

func NewThreadsOAuth(clientID, clientSecret, redirectURI string) *ThreadsOAuth {
	log.Printf("[ThreadsOAuth] Initializing with client_id=%s, redirect_uri=%s", clientID, redirectURI)
	return &ThreadsOAuth{
		config: &threads.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  redirectURI,
			Scopes:       []string{"threads_basic", "threads_content_publish"},
		},
	}
}

func (t *ThreadsOAuth) GenerateAuthURL(workspaceID string) string {
	client, _ := threads.NewClient(t.config)
	authURL := client.GetAuthURL(t.config.Scopes)

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
	log.Printf("[ThreadsOAuth] Config - ClientID: %s, RedirectURI: %s", t.config.ClientID, t.config.RedirectURI)

	client, err := threads.NewClient(t.config)
	if err != nil {
		log.Printf("[ThreadsOAuth] Failed to create client: %v", err)
		return nil, "", fmt.Errorf("failed to create threads client: %w", err)
	}

	log.Printf("[ThreadsOAuth] Calling ExchangeCodeForToken...")
	if err := client.ExchangeCodeForToken(ctx, code); err != nil {
		log.Printf("[ThreadsOAuth] ExchangeCodeForToken error: %v", err)
		log.Printf("[ThreadsOAuth] Error type: %T", err)
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}
	log.Printf("[ThreadsOAuth] ExchangeCodeForToken succeeded")

	log.Printf("[ThreadsOAuth] Calling GetLongLivedToken...")
	if err := client.GetLongLivedToken(ctx); err != nil {
		log.Printf("[ThreadsOAuth] GetLongLivedToken error: %v", err)
		return nil, "", fmt.Errorf("failed to get long-lived token: %w", err)
	}
	log.Printf("[ThreadsOAuth] GetLongLivedToken succeeded")

	tokenInfo := client.GetTokenInfo()
	if tokenInfo == nil {
		log.Printf("[ThreadsOAuth] GetTokenInfo returned nil")
		return nil, "", fmt.Errorf("failed to get token info")
	}

	log.Printf("[ThreadsOAuth] Got token info - UserID: %s, AccessToken length: %d", tokenInfo.UserID, len(tokenInfo.AccessToken))

	var expiresIn int
	if !tokenInfo.ExpiresAt.IsZero() {
		expiresIn = int(tokenInfo.ExpiresAt.Sub(tokenInfo.CreatedAt).Seconds())
	}

	return &TokenResponse{
		AccessToken: tokenInfo.AccessToken,
		ExpiresIn:   expiresIn,
		TokenType:   "bearer",
	}, tokenInfo.UserID, nil
}

func (t *ThreadsOAuth) RefreshToken(ctx context.Context, accessToken string) (*TokenResponse, error) {
	client, err := threads.NewClientWithToken(accessToken, t.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create threads client: %w", err)
	}

	if err := client.RefreshToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh thread token: %w", err)
	}

	tokenInfo := client.GetTokenInfo()
	if tokenInfo == nil {
		return nil, fmt.Errorf("failed to get token info after refresh")
	}

	var expiresIn int
	if !tokenInfo.ExpiresAt.IsZero() {
		expiresIn = int(tokenInfo.ExpiresAt.Sub(tokenInfo.CreatedAt).Seconds())
	}

	return &TokenResponse{
		AccessToken: tokenInfo.AccessToken,
		ExpiresIn:   expiresIn,
		TokenType:   "bearer",
	}, nil
}

func (t *ThreadsOAuth) GetProfile(ctx context.Context, accessToken, userID string) (*ThreadsProfile, error) {
	log.Printf("[ThreadsOAuth] GetProfile called for userID: %s", userID)

	client, err := threads.NewClientWithToken(accessToken, t.config)
	if err != nil {
		log.Printf("[ThreadsOAuth] GetProfile failed to create client: %v", err)
		return nil, fmt.Errorf("failed to create threads client: %w", err)
	}

	log.Printf("[ThreadsOAuth] Calling GetMe...")
	user, err := client.GetMe(ctx)
	if err != nil {
		log.Printf("[ThreadsOAuth] GetMe error: %v", err)
		return nil, fmt.Errorf("failed to get threads profile: %w", err)
	}

	log.Printf("[ThreadsOAuth] GetMe succeeded - ID: %s, Username: %s, Name: %s", user.ID, user.Username, user.Name)

	return &ThreadsProfile{
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
	}, nil
}

func (t *ThreadsOAuth) PublishTextPost(ctx context.Context, accessToken, userID, content string) (string, error) {
	cfg := &threads.Config{
		ClientID:     t.config.ClientID,
		ClientSecret: t.config.ClientSecret,
		RedirectURI:  t.config.RedirectURI,
		Scopes:       t.config.Scopes,
		RetryConfig: &threads.RetryConfig{
			MaxRetries:   0,
			InitialDelay: 0,
			MaxDelay:     0,
		},
	}

	client, err := threads.NewClientWithToken(accessToken, cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create threads client: %w", err)
	}

	post, err := client.CreateTextPost(ctx, &threads.TextPostContent{
		Text:            content,
		AutoPublishText: true,
	})
	if err != nil {
		if strings.Contains(err.Error(), "does not have permission") {
			log.Printf("[ThreadsOAuth] Post likely created but GetPost failed (this is OK - missing optional permission): %v", err)
			return "", nil
		}
		log.Printf("[ThreadsOAuth] CreateTextPost error: %v", err)
		return "", fmt.Errorf("failed to publish thread: %w", err)
	}

	log.Printf("[ThreadsOAuth] Successfully published thread with ID: %s", post.ID)
	return string(post.ID), nil
}

func (t *ThreadsOAuth) CreateMediaContainer(ctx context.Context, accessToken, userID, mediaType, content, mediaURL string) (string, error) {
	client, err := threads.NewClientWithToken(accessToken, t.config)
	if err != nil {
		return "", fmt.Errorf("failed to create threads client: %w", err)
	}

	containerID, err := client.CreateMediaContainer(ctx, mediaType, mediaURL, "")
	if err != nil {
		return "", fmt.Errorf("failed to create media container: %w", err)
	}

	return string(containerID), nil
}

func (t *ThreadsOAuth) PublishContainer(ctx context.Context, accessToken, userID, creationID string) (string, error) {
	return "", fmt.Errorf("use PublishTextPost or appropriate CreateXPost method instead")
}
