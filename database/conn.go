package database

import (
	"context"
	"database/sql"

	"github.com/alecthomas/errors"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*Queries
}

// Open database connection and return the associated typed DB wrapper.
func Open(ctx context.Context, dsn string) (*DB, error) {
	// Run migrations
	err := Migrate(ctx, dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to run migrations")
	}

	// Open connection to db
	conn, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open db")
	}
	return &DB{New(conn)}, nil
}
