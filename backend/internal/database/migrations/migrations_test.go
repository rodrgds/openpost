package migrations

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/openpost/backend/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

func TestRunMigrationsRemovesInactiveSetMemberships(t *testing.T) {
	t.Parallel()

	sqldb, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)

	db := bun.NewDB(sqldb, sqlitedialect.New())
	ctx := context.Background()

	for _, model := range []interface{}{
		(*models.Workspace)(nil),
		(*models.SocialAccount)(nil),
		(*models.SocialMediaSet)(nil),
		(*models.SocialMediaSetAccount)(nil),
	} {
		_, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		require.NoError(t, err)
	}

	accounts := []models.SocialAccount{
		{ID: "active-account", WorkspaceID: "ws-1", Platform: "x", AccountID: "1", AccessTokenEnc: []byte("token"), IsActive: true},
		{ID: "inactive-account", WorkspaceID: "ws-1", Platform: "x", AccountID: "2", AccessTokenEnc: []byte("token"), IsActive: true},
	}
	_, err = db.NewInsert().Model(&accounts).Exec(ctx)
	require.NoError(t, err)
	_, err = db.NewUpdate().Model((*models.SocialAccount)(nil)).Set("is_active = ?", false).Where("id = ?", "inactive-account").Exec(ctx)
	require.NoError(t, err)

	_, err = db.NewInsert().Model(&models.SocialMediaSet{ID: "set-1", WorkspaceID: "ws-1", Name: "Primary"}).Exec(ctx)
	require.NoError(t, err)

	rows := []models.SocialMediaSetAccount{
		{SetID: "set-1", SocialAccountID: "active-account"},
		{SetID: "set-1", SocialAccountID: "inactive-account"},
		{SetID: "set-1", SocialAccountID: "missing-account"},
	}
	_, err = db.NewInsert().Model(&rows).Exec(ctx)
	require.NoError(t, err)

	err = RunMigrations(db)
	require.NoError(t, err)

	var remaining []models.SocialMediaSetAccount
	err = db.NewSelect().Model(&remaining).Scan(ctx)
	require.NoError(t, err)
	require.Len(t, remaining, 1)
	require.Equal(t, "active-account", remaining[0].SocialAccountID)
}
