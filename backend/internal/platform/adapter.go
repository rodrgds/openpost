package platform

import (
	"context"
	"io"
)

// PublishRequest contains everything needed to publish a single post.
type PublishRequest struct {
	Content          string   // Post text content
	PlatformMediaIDs []string // Platform-specific media IDs from UploadMedia
	ReplyToID        string   // External ID of parent post (empty for first post in thread)
}

// UserProfile is a platform-agnostic user identity returned by GetProfile.
type UserProfile struct {
	ID          string
	Username    string
	DisplayName string
}

// TokenResult is a platform-agnostic token response.
type TokenResult struct {
	AccessToken  string            `json:"access_token"`
	RefreshToken string            `json:"refresh_token"`
	ExpiresIn    int               `json:"expires_in"`
	TokenType    string            `json:"token_type"`
	Extra        map[string]string `json:"extra"` // Platform-specific data (e.g., user ID for Threads)
}

// Adapter is the single interface every social platform must implement.
// This eliminates switch statements across publisher, token manager, and OAuth handlers.
//
// Each platform implementation lives in its own file (x.go, mastodon.go, etc.)
// and is registered in main.go via a map[string]Adapter.
type Adapter interface {
	// Auth flow
	// GenerateAuthURL returns the OAuth authorization URL.
	// extra contains platform-specific params (e.g. PKCE code_challenge for X).
	GenerateAuthURL(state string) (authURL string, extra map[string]string)

	// ExchangeCode exchanges an authorization code for tokens.
	// extra contains platform-specific params (e.g. PKCE verifier for X, server_name for Mastodon).
	ExchangeCode(ctx context.Context, code string, extra map[string]string) (*TokenResult, error)

	// RefreshToken refreshes an access token using the stored refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResult, error)

	// GetProfile fetches the authenticated user's profile.
	GetProfile(ctx context.Context, accessToken string) (*UserProfile, error)

	// Media upload — returns a platform-specific media ID (or URL for Threads).
	// The reader is consumed and should contain the raw file bytes.
	UploadMedia(ctx context.Context, accessToken, accountID, mimeType string, reader io.Reader) (string, error)

	// Publishing — returns an external ID for the published post.
	// For Bluesky this is JSON {"uri":"...","cid":"..."} for threading support.
	// For LinkedIn this is the activity URN for the first post, or comment ID for replies.
	Publish(ctx context.Context, accessToken, accountID string, req *PublishRequest) (string, error)
}
