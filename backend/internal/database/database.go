package database

import (
	"context"
	"database/sql"

	"github.com/openpost/backend/internal/models"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/sqliteshim"
)

func InitDB(dsn string) (*bun.DB, error) {
	// DSN e.g. "file:openpost.db?cache=shared&mode=rwc"
	sqldb, err := sql.Open(sqliteshim.ShimName, dsn)
	if err != nil {
		return nil, err
	}

	// SQLite highly recommends max open conns to 1 when writing is involved
	// though WAL mode helps with concurrent readers
	sqldb.SetMaxOpenConns(1)

	db := bun.NewDB(sqldb, sqlitedialect.New())

	// Performance PRAGMAs
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000;"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, err
	}

	return db, nil
}

func CreateSchema(db *bun.DB) error {
	ctx := context.Background()
	m := []interface{}{
		(*models.Workspace)(nil),
		(*models.User)(nil),
		(*models.WorkspaceMember)(nil),
		(*models.SocialAccount)(nil),
		(*models.XOAuthRequestToken)(nil),
		(*models.Post)(nil),
		(*models.PostDestination)(nil),
		(*models.MediaAttachment)(nil),
		(*models.PostMedia)(nil),
		(*models.Job)(nil),
		(*models.SocialMediaSet)(nil),
		(*models.SocialMediaSetAccount)(nil),
		(*models.PostVariant)(nil),
		(*models.PostingSchedule)(nil),
		(*models.Prompt)(nil),
	}
	for _, model := range m {
		if _, err := db.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}
