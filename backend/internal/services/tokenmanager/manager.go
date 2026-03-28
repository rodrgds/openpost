package tokenmanager

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/openpost/backend/internal/services/oauth"
)

type TokenManager struct {
	db       *bun.DB
	crypto   *crypto.TokenEncryptor
	twitter  *oauth.TwitterOAuth
	linkedin *oauth.LinkedInOAuth
	threads  *oauth.ThreadsOAuth
	bluesky  *oauth.BlueskyOAuth
	mastodon map[string]*oauth.MastodonOAuth
}

func NewTokenManager(db *bun.DB, encryptor *crypto.TokenEncryptor) *TokenManager {
	return &TokenManager{
		db:     db,
		crypto: encryptor,
	}
}

// SetTwitterOAuth sets the Twitter OAuth provider for token refresh
func (tm *TokenManager) SetTwitterOAuth(tw *oauth.TwitterOAuth) {
	tm.twitter = tw
}

// SetLinkedInOAuth sets the LinkedIn OAuth provider for token refresh
func (tm *TokenManager) SetLinkedInOAuth(li *oauth.LinkedInOAuth) {
	tm.linkedin = li
}

// SetThreadsOAuth sets the Threads OAuth provider for token refresh
func (tm *TokenManager) SetThreadsOAuth(th *oauth.ThreadsOAuth) {
	tm.threads = th
}

// SetBlueskyOAuth sets the Bluesky OAuth provider for token refresh
func (tm *TokenManager) SetBlueskyOAuth(bs *oauth.BlueskyOAuth) {
	tm.bluesky = bs
}

// SetMastodonOAuth sets the Mastodon OAuth providers for token refresh
func (tm *TokenManager) SetMastodonOAuth(servers map[string]*oauth.MastodonOAuth) {
	tm.mastodon = servers
}

// GetValidAccessToken returns a decrypted access token, automatically refreshing if close to expiry
func (tm *TokenManager) GetValidAccessToken(ctx context.Context, accountID string) (string, error) {
	account := new(models.SocialAccount)
	err := tm.db.NewSelect().
		Model(account).
		Where("id = ?", accountID).
		Scan(ctx)

	if err != nil {
		return "", err
	}

	if !account.IsActive {
		return "", fmt.Errorf("account is disconnected: %s", account.ErrorMessage)
	}

	// Mastodon tokens have empty IsZero time, so they skip this check
	// Mastodon tokens don't expire (they're valid until revoked)
	if account.TokenExpiresAt.IsZero() {
		return tm.crypto.Decrypt(account.AccessTokenEnc)
	}

	// Refresh token if it expires within 5 minutes
	if account.TokenExpiresAt.Before(time.Now().Add(5 * time.Minute)) {
		return tm.refreshToken(ctx, account)
	}

	return tm.crypto.Decrypt(account.AccessTokenEnc)
}

func (tm *TokenManager) refreshToken(ctx context.Context, account *models.SocialAccount) (string, error) {
	if len(account.RefreshTokenEnc) == 0 {
		return "", fmt.Errorf("no refresh token available for account %s", account.ID)
	}

	refreshToken, err := tm.crypto.Decrypt(account.RefreshTokenEnc)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	var tokenResp *oauth.TokenResponse

	switch account.Platform {
	case "x":
		if tm.twitter == nil {
			return "", fmt.Errorf("twitter OAuth not configured")
		}
		tokenResp, err = tm.twitter.RefreshToken(ctx, refreshToken)

	case "linkedin":
		if tm.linkedin == nil {
			return "", fmt.Errorf("linkedin OAuth not configured")
		}
		tokenResp, err = tm.linkedin.RefreshToken(ctx, refreshToken)

	case "threads":
		if tm.threads == nil {
			return "", fmt.Errorf("threads OAuth not configured")
		}
		tokenResp, err = tm.threads.RefreshToken(ctx, refreshToken)

	case "bluesky":
		if tm.bluesky == nil {
			return "", fmt.Errorf("bluesky not configured")
		}
		var bsSession *oauth.BlueskySession
		bsSession, err = tm.bluesky.RefreshSession(ctx, refreshToken)
		if bsSession != nil {
			tokenResp = &oauth.TokenResponse{
				AccessToken:  bsSession.AccessToken,
				RefreshToken: bsSession.RefreshToken,
			}
		}

	case "mastodon":
		// Mastodon tokens don't expire - they stay valid until revoked
		return tm.crypto.Decrypt(account.AccessTokenEnc)

	default:
		return "", fmt.Errorf("unsupported platform for token refresh: %s", account.Platform)
	}

	if err != nil {
		log.Printf("[TokenManager] Failed to refresh token for %s account %s: %v", account.Platform, account.ID, err)
		// Mark account as having an error but don't deactivate it
		_, _ = tm.db.NewUpdate().Model(account).
			Set("error_message = ?", err.Error()).
			Where("id = ?", account.ID).
			Exec(ctx)
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

	// Encrypt and store new tokens
	encAccess, err := tm.crypto.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt access token: %w", err)
	}

	var encRefresh []byte
	if tokenResp.RefreshToken != "" {
		encRefresh, err = tm.crypto.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	// Update account with new tokens
	account.AccessTokenEnc = encAccess
	if len(encRefresh) > 0 {
		account.RefreshTokenEnc = encRefresh
	}
	account.TokenExpiresAt = expiresAt
	account.ErrorMessage = ""

	_, err = tm.db.NewUpdate().Model(account).
		Set("access_token_encrypted = ?", account.AccessTokenEnc).
		Set("refresh_token_encrypted = ?", account.RefreshTokenEnc).
		Set("token_expires_at = ?", account.TokenExpiresAt).
		Set("error_message = ?", "").
		Where("id = ?", account.ID).
		Exec(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to update account tokens: %w", err)
	}

	log.Printf("[TokenManager] Successfully refreshed token for %s account %s", account.Platform, account.ID)
	return tokenResp.AccessToken, nil
}
