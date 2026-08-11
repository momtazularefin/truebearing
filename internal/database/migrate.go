package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var migrationFS embed.FS

// Migrate applies all pending .up.sql migrations embedded in the binary.
// It creates a schema_migrations tracking table if it does not exist, then
// executes each unapplied migration inside a transaction.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	// Ensure the tracking table exists.
	const createTable = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`
	if _, err := pool.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("migrate: create schema_migrations table: %w", err)
	}

	// Read and sort embedded migration files.
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("migrate: read embedded migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	// Apply each migration that has not yet been recorded.
	for _, name := range names {
		version := strings.TrimSuffix(name, filepath.Ext(name))
		version = strings.TrimSuffix(version, ".up") // e.g. "001_init"

		// Check if already applied.
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("migrate: check version %s: %w", version, err)
		}
		if exists {
			slog.Info("migration already applied, skipping", "version", version)
			continue
		}

		// Read SQL content.
		sql, err := migrationFS.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("migrate: read file %s: %w", name, err)
		}

		// Execute in a transaction.
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate: begin tx for %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: exec %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", version,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: record version %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", version, err)
		}

		slog.Info("migration applied", "version", version)
	}

	return nil
}
