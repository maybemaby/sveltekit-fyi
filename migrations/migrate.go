package migrations

import (
	"context"
	"database/sql"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var migrations embed.FS

func RunMigrations(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(migrations)

	err := goose.SetDialect("sqlite3")

	if err != nil {
		return err
	}

	return goose.UpContext(ctx, db, ".")
}
