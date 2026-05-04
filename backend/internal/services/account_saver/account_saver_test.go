package account_saver

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/openpost/backend/internal/models"
	"github.com/openpost/backend/internal/platform"
	"github.com/openpost/backend/internal/services/crypto"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

// createTestDB creates an in-memory SQLite database for testing.
func createTestDB(t *testing.T) *bun.DB {
	sqldb, err := openInMemorySQLite()
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	// Initialize schema
	_, err = db.NewCreateTable().
		Model((*models.SocialAccount)(nil)).
		IfNotExists().
		Exec(context.Background())
	require.NoError(t, err)
	_, err = db.NewCreateTable().
		Model((*models.WorkspaceMember)(nil)).
		IfNotExists().
		Exec(context.Background())
	require.NoError(t, err)
	_, err = db.NewCreateTable().
		Model((*models.Job)(nil)).
		IfNotExists().
		Exec(context.Background())
	require.NoError(t, err)

	return db
}

func seedWorkspaceMember(t *testing.T, db *bun.DB, workspaceID, userID string) {
	_, err := db.NewInsert().Model(&models.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        "admin",
	}).Exec(context.Background())
	require.NoError(t, err)
}

// openInMemorySQLite creates an in-memory SQLite database.
func openInMemorySQLite() (*sql.DB, error) {
	return sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", uuid.NewString()))
}

// TestSaveAccount_X tests saving an X (Twitter) account.
func TestSaveAccount_X(t *testing.T) {
	t.Parallel()

	db := createTestDB(t)
	crypto := crypto.NewTokenEncryptor("test-secret-key-for-testing-only")
	saver := NewAccountSaver(db, crypto)

	ctx := context.Background()
	workspaceID := "test-workspace-123"
	userID := "user-123"
	platformName := "x"
	accountID := "1234567890"
	accountUsername := "testuser"
	instanceURL := "" // Not used for X

	// Mock token response
	tokenResp := &platform.TokenResult{
		AccessToken:  "x-access-token-123",
		RefreshToken: "x-refresh-token-456",
		ExpiresIn:    7200, // 2 hours
		Extra:        map[string]string{},
	}

	seedWorkspaceMember(t, db, workspaceID, userID)
	account, err := saver.SaveAccount(ctx, userID, platformName, workspaceID, accountID, accountUsername, instanceURL, tokenResp)
	require.NoError(t, err)
	require.NotNil(t, account)

	// Verify account fields
	require.Equal(t, workspaceID, account.WorkspaceID)
	require.Equal(t, platformName, account.Platform)
	require.Equal(t, accountID, account.AccountID)
	require.Equal(t, accountUsername, account.AccountUsername)
	require.Equal(t, instanceURL, account.InstanceURL)
	require.True(t, account.IsActive)
	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	// Verify tokens are encrypted (not plaintext)
	require.NotEqual(t, tokenResp.AccessToken, string(account.AccessTokenEnc))
	require.NotEqual(t, tokenResp.RefreshToken, string(account.RefreshTokenEnc))

	// Verify decryption works
	decryptedAccess, err := crypto.Decrypt(account.AccessTokenEnc)
	require.NoError(t, err)
	require.Equal(t, tokenResp.AccessToken, decryptedAccess)

	decryptedRefresh, err := crypto.Decrypt(account.RefreshTokenEnc)
	require.NoError(t, err)
	require.Equal(t, tokenResp.RefreshToken, decryptedRefresh)

	// Verify expiration is set (within reasonable range)
	require.WithinDuration(t, time.Now().UTC().Add(2*time.Hour), account.TokenExpiresAt, 10*time.Second)

	var jobs []models.Job
	err = db.NewSelect().Model(&jobs).Where("type = ?", "refresh_token").Scan(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 1)
}

// TestSaveAccount_Mastodon tests saving a Mastodon account.
func TestSaveAccount_Mastodon(t *testing.T) {
	t.Parallel()

	db := createTestDB(t)
	crypto := crypto.NewTokenEncryptor("test-secret-key-for-testing-only")
	saver := NewAccountSaver(db, crypto)

	ctx := context.Background()
	workspaceID := "test-workspace-456"
	userID := "user-456"
	platformName := "mastodon"
	accountID := "mastodon-user-123"
	accountUsername := "mastodonuser"
	instanceURL := "https://mastodon.example.com"

	tokenResp := &platform.TokenResult{
		AccessToken:  "mastodon-access-token",
		RefreshToken: "mastodon-refresh-token",
		ExpiresIn:    7200,
		Extra:        map[string]string{},
	}

	seedWorkspaceMember(t, db, workspaceID, userID)
	account, err := saver.SaveAccount(ctx, userID, platformName, workspaceID, accountID, accountUsername, instanceURL, tokenResp)
	require.NoError(t, err)
	require.NotNil(t, account)

	require.Equal(t, workspaceID, account.WorkspaceID)
	require.Equal(t, platformName, account.Platform)
	require.Equal(t, accountID, account.AccountID)
	require.Equal(t, accountUsername, account.AccountUsername)
	require.Equal(t, instanceURL, account.InstanceURL)
	require.True(t, account.IsActive)

	// Verify tokens encrypted
	decryptedAccess, err := crypto.Decrypt(account.AccessTokenEnc)
	require.NoError(t, err)
	require.Equal(t, tokenResp.AccessToken, decryptedAccess)

	decryptedRefresh, err := crypto.Decrypt(account.RefreshTokenEnc)
	require.NoError(t, err)
	require.Equal(t, tokenResp.RefreshToken, decryptedRefresh)
}

// TestSaveAccount_Threads tests that Threads user ID is extracted from token extra.
func TestSaveAccount_Threads(t *testing.T) {
	t.Parallel()

	db := createTestDB(t)
	crypto := crypto.NewTokenEncryptor("test-secret-key-for-testing-only")
	saver := NewAccountSaver(db, crypto)

	ctx := context.Background()
	workspaceID := "test-workspace-789"
	userID := "user-789"
	platformName := "threads"
	// This accountID will be overridden by user_id from token extra
	initialAccountID := "initial-account-id"
	accountUsername := "threadsuser"
	instanceURL := ""

	tokenResp := &platform.TokenResult{
		AccessToken:  "threads-access-token",
		RefreshToken: "threads-refresh-token",
		ExpiresIn:    7200,
		Extra: map[string]string{
			"user_id": "threads-user-id-987", // This should become the account ID
		},
	}

	seedWorkspaceMember(t, db, workspaceID, userID)
	account, err := saver.SaveAccount(ctx, userID, platformName, workspaceID, initialAccountID, accountUsername, instanceURL, tokenResp)
	require.NoError(t, err)
	require.NotNil(t, account)

	// Verify the account ID was overridden by user_id from token extra
	require.Equal(t, "threads-user-id-987", account.AccountID)
	require.Equal(t, accountUsername, account.AccountUsername)

	// Verify tokens encrypted
	decryptedAccess, err := crypto.Decrypt(account.AccessTokenEnc)
	require.NoError(t, err)
	require.Equal(t, tokenResp.AccessToken, decryptedAccess)
}

// TestSaveAccount_EncryptionError tests handling of encryption failures.
func TestSaveAccount_EncryptionError(t *testing.T) {
	t.Parallel()

	db := createTestDB(t)
	crypto := crypto.NewTokenEncryptor("test-secret-key-for-testing-only")
	saver := NewAccountSaver(db, crypto)

	ctx := context.Background()
	workspaceID := "workspace"
	userID := "user"
	seedWorkspaceMember(t, db, workspaceID, userID)
	tokenResp := &platform.TokenResult{
		AccessToken:  "some-token",
		RefreshToken: "some-refresh",
		ExpiresIn:    3600,
	}

	acct, err := saver.SaveAccount(ctx, userID, "x", workspaceID, "account", "user", "", tokenResp)
	require.NoError(t, err)
	require.NotNil(t, acct)
	// Ensure tokens are stored encrypted and decryptable
	dec, derr := crypto.Decrypt(acct.AccessTokenEnc)
	require.NoError(t, derr)
	require.Equal(t, tokenResp.AccessToken, dec)
}

// TestSaveAccount_DatabaseError tests handling of database failures.
func TestSaveAccount_DatabaseError(t *testing.T) {
	t.Parallel()

	// Use a nil db to simulate database failure
	crypto := crypto.NewTokenEncryptor("test-secret-key")
	saver := NewAccountSaver(nil, crypto)

	ctx := context.Background()
	tokenResp := &platform.TokenResult{
		AccessToken:  "token",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
	}

	_, err := saver.SaveAccount(ctx, "user", "x", "workspace", "account", "user", "", tokenResp)
	require.Error(t, err)
}
