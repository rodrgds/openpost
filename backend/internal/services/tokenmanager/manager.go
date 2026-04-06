package tokenmanager

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/uptrace/bun"

	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/platform"
	"github.com/openpost/backend/internal/services/crypto"
)

type TokenManager struct {
	db        *bun.DB
	crypto    *crypto.TokenEncryptor
	providers map[string]platform.PlatformAdapter
}

func NewTokenManager(db *bun.DB, encryptor *crypto.TokenEncryptor) *TokenManager {
	return &TokenManager{
		db:        db,
		crypto:    encryptor,
		providers: make(map[string]platform.PlatformAdapter),
	}
}

func (tm *TokenManager) SetProvider(platformName string, adapter platform.PlatformAdapter) {
	tm.providers[platformName] = adapter
}

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

	if account.TokenExpiresAt.IsZero() {
		return tm.crypto.Decrypt(account.AccessTokenEnc)
	}

	if account.TokenExpiresAt.Before(time.Now().UTC().Add(5 * time.Minute)) {
		return tm.refreshToken(ctx, account)
	}

	return tm.crypto.Decrypt(account.AccessTokenEnc)
}

func (tm *TokenManager) ForceRefreshAccessToken(ctx context.Context, accountID string) (string, error) {
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

	if len(account.RefreshTokenEnc) == 0 {
		return "", fmt.Errorf("no refresh token available for account %s", account.ID)
	}

	return tm.refreshToken(ctx, account)
}

func (tm *TokenManager) refreshToken(ctx context.Context, account *models.SocialAccount) (string, error) {
	providerKey := account.Platform
	if account.Platform == "mastodon" {
		providerKey = "mastodon:" + account.InstanceURL
	}

	provider, ok := tm.providers[providerKey]
	if !ok {
		return "", fmt.Errorf("unsupported platform for token refresh: %s (instance: %s)", account.Platform, account.InstanceURL)
	}

	if len(account.RefreshTokenEnc) == 0 {
		return "", fmt.Errorf("no refresh token available for account %s", account.ID)
	}

	refreshToken, err := tm.crypto.Decrypt(account.RefreshTokenEnc)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	tokenResp, err := provider.RefreshToken(ctx, refreshToken)
	if err != nil {
		log.Printf("[TokenManager] Failed to refresh token for %s account %s: %v", account.Platform, account.ID, err)
		_, _ = tm.db.NewUpdate().Model(account).
			Set("error_message = ?", err.Error()).
			Where("id = ?", account.ID).
			Exec(ctx)
		return "", fmt.Errorf("failed to refresh token: %w", err)
	}

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
		expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	_, err = tm.db.NewUpdate().Model(account).
		Set("access_token_encrypted = ?", encAccess).
		Set("refresh_token_encrypted = ?", encRefresh).
		Set("token_expires_at = ?", expiresAt).
		Set("error_message = ?", "").
		Where("id = ?", account.ID).
		Exec(ctx)

	if err != nil {
		return "", fmt.Errorf("failed to update account tokens: %w", err)
	}

	log.Printf("[TokenManager] Successfully refreshed token for %s account %s", account.Platform, account.ID)
	return tokenResp.AccessToken, nil
}
