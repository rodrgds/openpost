package tokenmanager

import (
	"context"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/services/crypto"
)

type TokenManager struct {
	db     *bun.DB
	crypto *crypto.TokenEncryptor
}

func NewTokenManager(db *bun.DB, encryptor *crypto.TokenEncryptor) *TokenManager {
	return &TokenManager{
		db:     db,
		crypto: encryptor,
	}
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

	// Mastodon tokens have empty IsZero time, so it skips this lock automatically
	if !account.TokenExpiresAt.IsZero() && account.TokenExpiresAt.Before(time.Now().Add(5*time.Minute)) {
		return tm.refreshToken(ctx, account)
	}

	return tm.crypto.Decrypt(account.AccessTokenEnc)
}

func (tm *TokenManager) refreshToken(ctx context.Context, account *models.SocialAccount) (string, error) {
	// Stub to be implemented per-platform
	return "", fmt.Errorf("refresh token logic not implemented for platform %s", account.Platform)
}
