// Package database provides PostgreSQL connection and migration utilities.
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Connect creates a new pgxpool connection pool, verifies connectivity with a
// ping, and returns the pool. The caller is responsible for calling pool.Close.
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: unable to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: unable to ping: %w", err)
	}

	return pool, nil
}
