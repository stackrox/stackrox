package vulnimporter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore/datastore"
	ccpostgres "github.com/quay/claircore/datastore/postgres"
)

// NewMatcherStore connects to Clair's PostgreSQL and returns a MatcherStore
// suitable for importing vulnerability data directly.
//
// doMigration is false because Clair manages its own schema migrations.
// If Clair hasn't started yet and tables don't exist, this will retry
// until the context is canceled or the connection succeeds.
func NewMatcherStore(ctx context.Context, connString string) (datastore.MatcherStore, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing Clair DB connection string: %w", err)
	}
	cfg.ConnConfig.RuntimeParams["application_name"] = "clair-adapter-importer"

	var pool *pgxpool.Pool
	for attempt := 1; attempt <= 30; attempt++ {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				break
			}
			pool.Close()
		}
		slog.WarnContext(ctx, "waiting for Clair database", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Second):
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Clair database after 30 attempts: %w", err)
	}

	// Wait for Clair's tables to exist (Clair runs migrations on startup).
	for attempt := 1; attempt <= 30; attempt++ {
		var exists bool
		err = pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'update_operation')`).Scan(&exists)
		if err == nil && exists {
			break
		}
		slog.WarnContext(ctx, "waiting for Clair schema (update_operation table)", "attempt", attempt)
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	store, err := ccpostgres.InitPostgresMatcherStore(ctx, pool, false)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("initializing Clair matcher store: %w", err)
	}

	return store, nil
}
