package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenantTx executes the provided function within a PostgreSQL transaction bound
// to the specified tenant ID via `set_config('app.tenant_id', $1, true)`.
//
// The third argument `true` is load-bearing (D004): it ensures the configuration
// is local to the current transaction and automatically resets when the transaction
// completes, preventing connection pool leaks between requests.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Bind tenant context to this specific transaction.
	_, err = tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String())
	if err != nil {
		return fmt.Errorf("failed to bind tenant to transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tenant transaction: %w", err)
	}

	return nil
}

// QueueTenantBinding queues a transaction-scoped tenant binding as the first
// statement in a pgx Batch, allowing queries to be pipelined in a single network round-trip.
func QueueTenantBinding(batch *pgx.Batch, tenantID uuid.UUID) {
	batch.Queue("SELECT set_config('app.tenant_id', $1, true)", tenantID.String())
}
