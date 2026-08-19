package database

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.up.sql
var migrationFS embed.FS

// migration is one embedded .up.sql file, already read and named.
type migration struct {
	// version is the tracking key recorded in schema_migrations,
	// e.g. "001_create_tenants".
	version string
	// sql is the statement text of the migration.
	sql string
}

// loadMigrations reads every embedded .up.sql file in lexical order and derives
// each one's tracking version.
//
// It uses "path" rather than "path/filepath" throughout. Keys in an embed.FS
// are always slash-separated regardless of host operating system, so joining
// with filepath produces a backslash on Windows and the lookup misses.
func loadMigrations() ([]migration, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, fmt.Errorf("migrate: read embedded migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	migrations := make([]migration, 0, len(names))
	for _, name := range names {
		sql, err := migrationFS.ReadFile(path.Join("migrations", name))
		if err != nil {
			return nil, fmt.Errorf("migrate: read file %s: %w", name, err)
		}

		version := strings.TrimSuffix(name, path.Ext(name))
		version = strings.TrimSuffix(version, ".up") // e.g. "001_create_tenants"

		migrations = append(migrations, migration{version: version, sql: string(sql)})
	}

	return migrations, nil
}

// Migrate applies all pending .up.sql migrations embedded in the binary.
// It creates a schema_migrations tracking table if it does not exist, then
// executes each unapplied migration inside a transaction.
//
// This runs at server boot only for the single-container local stack. A
// multi-replica deployment uses the dedicated migrate role instead, because
// concurrent boot-time migration races between replicas.
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

	// Read every embedded migration up front, so a packaging error fails
	// before any statement is applied rather than halfway through.
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}

	for _, m := range migrations {
		// Check if already applied.
		var exists bool
		err := pool.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", m.version,
		).Scan(&exists)
		if err != nil {
			return fmt.Errorf("migrate: check version %s: %w", m.version, err)
		}
		if exists {
			slog.Info("migration already applied, skipping", "version", m.version)
			continue
		}

		// Execute in a transaction.
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrate: begin tx for %s: %w", m.version, err)
		}

		if _, err := tx.Exec(ctx, m.sql); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: exec %s: %w", m.version, err)
		}

		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)", m.version,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrate: record version %s: %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", m.version, err)
		}

		slog.Info("migration applied", "version", m.version)
	}

	return nil
}
