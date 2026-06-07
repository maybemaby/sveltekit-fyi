package internal

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

func ConnectDB(ctx context.Context) (*sql.DB, error) {
	dbPath := "sveltekit_fyi.db?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"

	db, err := sql.Open("sqlite", dbPath)

	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
