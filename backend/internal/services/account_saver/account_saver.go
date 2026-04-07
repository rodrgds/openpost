package account_saver

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/platform"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/uptrace/bun"
)

// AccountSaver handles saving social account information to the database.
// This service extracts the duplicated account-saving logic from the OAuth handler.
type AccountSaver struct {
	db     *bun.DB
	crypto *crypto.TokenEncryptor
}

// NewAccountSaver creates a new AccountSaver instance.
func NewAccountSaver(db *bun.DB, crypto *crypto.TokenEncryptor) *AccountSaver {
	return &AccountSaver{
		db:     db,
		crypto: crypto,
	}
}

// SaveAccount saves a social account with encrypted tokens.
// It handles the common logic of extracting account info, encrypting tokens,
// and inserting into the social_accounts table.
func (s *AccountSaver) SaveAccount(ctx context.Context, platformName, workspaceID, accountID, accountUsername, instanceURL string, tokenResp *platform.TokenResult) (*models.SocialAccount, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	// For Threads, the account ID comes from the token response extra
	if tokenResp.Extra != nil {
		if uid, ok := tokenResp.Extra["user_id"]; ok && uid != "" {
			accountID = uid
		}
	}

	encAccess, err := s.crypto.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return nil, err
	}

	var encRefresh []byte
	if tokenResp.RefreshToken != "" {
		encRefresh, err = s.crypto.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			return nil, err
		}
	}

	var expiresAt time.Time
	if tokenResp.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	}

	account := &models.SocialAccount{
		ID:              uuid.New().String(),
		WorkspaceID:     workspaceID,
		Platform:        platformName,
		AccountID:       accountID,
		AccountUsername: accountUsername,
		InstanceURL:     instanceURL,
		AccessTokenEnc:  encAccess,
		RefreshTokenEnc: encRefresh,
		TokenExpiresAt:  expiresAt,
		IsActive:        true,
		CreatedAt:       time.Now().UTC(),
	}

	if _, err := s.db.NewInsert().Model(account).Exec(ctx); err != nil {
		return nil, err
	}

	return account, nil
}
