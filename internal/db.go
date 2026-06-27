package internal

import (
	"context"
	"database/sql"
	"errors"
	"os"

	_ "modernc.org/sqlite"
)

func ConnectDB(ctx context.Context) (*sql.DB, error) {
	dbPath := os.Getenv("DATABASE_URL")

	if dbPath == "" {
		return nil, errors.New("DATABASE_URL environment variable is empty")
	}

	db, err := sql.Open("sqlite", dbPath)

	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func ConnectDBReadOnly(ctx context.Context) (*sql.DB, error) {
	dbPath := os.Getenv("DATABASE_URL")

	if dbPath == "" {
		return nil, errors.New("DATABASE_URL environment variable is empty")
	}

	db, err := sql.Open("sqlite", dbPath)

	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}
