package oauth

import (
	"context"
	"fmt"

	threads "github.com/tirthpatell/threads-go"
)

type ThreadsProfile struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type ThreadsOAuth struct {
	config *threads.Config
}

func NewThreadsOAuth(clientID, clientSecret, redirectURI string) *ThreadsOAuth {
	return &ThreadsOAuth{
		config: &threads.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURI:  redirectURI,
			Scopes:       []string{"threads_basic", "threads_content_publish"},
		},
	}
}

func (t *ThreadsOAuth) GenerateAuthURL(state string) string {
	client, _ := threads.NewClient(t.config)
	return client.GetAuthURL(t.config.Scopes)
}

func (t *ThreadsOAuth) ExchangeCode(ctx context.Context, code string) (*TokenResponse, string, error) {
	client, err := threads.NewClient(t.config)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create threads client: %w", err)
	}

	if err := client.ExchangeCodeForToken(ctx, code); err != nil {
		return nil, "", fmt.Errorf("failed to exchange code: %w", err)
	}

	if err := client.GetLongLivedToken(ctx); err != nil {
		return nil, "", fmt.Errorf("failed to get long-lived token: %w", err)
	}

	tokenInfo := client.GetTokenInfo()
	if tokenInfo == nil {
		return nil, "", fmt.Errorf("failed to get token info")
	}

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
	client, err := threads.NewClientWithToken(accessToken, t.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create threads client: %w", err)
	}

	user, err := client.GetMe(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get threads profile: %w", err)
	}

	return &ThreadsProfile{
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
	}, nil
}

func (t *ThreadsOAuth) PublishTextPost(ctx context.Context, accessToken, userID, content string) (string, error) {
	client, err := threads.NewClientWithToken(accessToken, t.config)
	if err != nil {
		return "", fmt.Errorf("failed to create threads client: %w", err)
	}

	post, err := client.CreateTextPost(ctx, &threads.TextPostContent{
		Text: content,
	})
	if err != nil {
		return "", fmt.Errorf("failed to publish thread: %w", err)
	}

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
