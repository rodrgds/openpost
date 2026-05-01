package handlers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/openpost/backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func createHandlerTestDB(t *testing.T, modelsToCreate ...interface{}) *bun.DB {
	t.Helper()

	sqldb, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name()))
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	for _, model := range modelsToCreate {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(context.Background())
		require.NoError(t, err)
	}

	return db
}

func TestSetHandlerValidateAccountsBelongToWorkspaceRejectsInactiveOrForeignAccounts(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t, (*models.SocialAccount)(nil))
	handler := &SetHandler{db: db}
	ctx := context.Background()

	accounts := []models.SocialAccount{
		{ID: "active-in-workspace", WorkspaceID: "ws-1", Platform: "x", AccountID: "1", AccessTokenEnc: []byte("token"), IsActive: true},
		{ID: "inactive-in-workspace", WorkspaceID: "ws-1", Platform: "x", AccountID: "2", AccessTokenEnc: []byte("token"), IsActive: true},
		{ID: "active-in-other-workspace", WorkspaceID: "ws-2", Platform: "x", AccountID: "3", AccessTokenEnc: []byte("token"), IsActive: true},
	}
	_, err := db.NewInsert().Model(&accounts).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().Model((*models.SocialAccount)(nil)).Set("is_active = ?", false).Where("id = ?", "inactive-in-workspace").Exec(ctx)
	require.NoError(t, err)

	require.NoError(t, handler.validateAccountsBelongToWorkspace(ctx, "ws-1", []string{"active-in-workspace"}))
	require.Error(t, handler.validateAccountsBelongToWorkspace(ctx, "ws-1", []string{"inactive-in-workspace"}))
	require.Error(t, handler.validateAccountsBelongToWorkspace(ctx, "ws-1", []string{"active-in-other-workspace"}))
}

func TestSetHandlerLoadSetAccountsOnlyReturnsActiveAccounts(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t,
		(*models.SocialAccount)(nil),
		(*models.SocialMediaSet)(nil),
		(*models.SocialMediaSetAccount)(nil),
	)
	handler := &SetHandler{db: db}
	ctx := context.Background()

	accounts := []models.SocialAccount{
		{ID: "active-account", WorkspaceID: "ws-1", Platform: "x", AccountID: "1", AccountUsername: "active", AccessTokenEnc: []byte("token"), IsActive: true},
		{ID: "inactive-account", WorkspaceID: "ws-1", Platform: "threads", AccountID: "2", AccountUsername: "inactive", AccessTokenEnc: []byte("token"), IsActive: true},
	}
	_, err := db.NewInsert().Model(&accounts).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().Model((*models.SocialAccount)(nil)).Set("is_active = ?", false).Where("id = ?", "inactive-account").Exec(ctx)
	require.NoError(t, err)

	sets := []models.SocialMediaSet{{ID: "set-1", WorkspaceID: "ws-1", Name: "Primary"}}
	_, err = db.NewInsert().Model(&sets).Exec(ctx)
	require.NoError(t, err)

	setAccounts := []models.SocialMediaSetAccount{
		{SetID: "set-1", SocialAccountID: "active-account"},
		{SetID: "set-1", SocialAccountID: "inactive-account"},
	}
	_, err = db.NewInsert().Model(&setAccounts).Exec(ctx)
	require.NoError(t, err)

	loaded, err := handler.loadSingleSetAccounts(ctx, "set-1")
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Equal(t, "active-account", loaded[0].SocialAccountID)
	require.Equal(t, "active", loaded[0].AccountUsername)
}

func TestPostHandlerValidateAccountsBelongToWorkspaceRejectsInactiveAccounts(t *testing.T) {
	t.Parallel()

	db := createHandlerTestDB(t, (*models.SocialAccount)(nil))
	handler := &PostHandler{db: db}
	ctx := context.Background()

	accounts := []models.SocialAccount{
		{ID: "active-account", WorkspaceID: "ws-1", Platform: "x", AccountID: "1", AccessTokenEnc: []byte("token"), IsActive: true},
		{ID: "inactive-account", WorkspaceID: "ws-1", Platform: "x", AccountID: "2", AccessTokenEnc: []byte("token"), IsActive: true},
	}
	_, err := db.NewInsert().Model(&accounts).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().Model((*models.SocialAccount)(nil)).Set("is_active = ?", false).Where("id = ?", "inactive-account").Exec(ctx)
	require.NoError(t, err)

	require.NoError(t, handler.validateAccountsBelongToWorkspace(ctx, "ws-1", []string{"active-account"}))
	require.Error(t, handler.validateAccountsBelongToWorkspace(ctx, "ws-1", []string{"inactive-account"}))
}
