package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quay/claircore"
	"github.com/quay/claircore/datastore"
	"github.com/quay/claircore/datastore/postgres"
)

// MatcherStore represents a matcher datastore.
// It is a wrapper around datastore.MatcherStore with some added StackRox-specific functionality.
//
//go:generate mockgen-wrapper
type MatcherStore interface {
	datastore.MatcherStore

	Distributions(ctx context.Context) ([]claircore.Distribution, error)

	// ReindexVulnTables rebuilds indexes on the vuln and uo_vuln tables.
	// This repairs index corruption that can cause persistent FK violations
	// during vulnerability imports.
	ReindexVulnTables(ctx context.Context) error
}

type matcherStore struct {
	datastore.MatcherStore
	pool *pgxpool.Pool
}

// InitPostgresMatcherStore initializes a matcher datastore.
func InitPostgresMatcherStore(ctx context.Context, pool *pgxpool.Pool, doMigration bool) (MatcherStore, error) {
	if pool == nil {
		return nil, errors.New("pool must be non-nil")
	}

	store, err := postgres.InitPostgresMatcherStore(ctx, pool, doMigration)
	if err != nil {
		return nil, err
	}

	return &matcherStore{
		MatcherStore: store,
		pool:         pool,
	}, nil
}

func (m *matcherStore) ReindexVulnTables(ctx context.Context) error {
	for _, table := range []string{"vuln", "uo_vuln"} {
		if _, err := m.pool.Exec(ctx, "REINDEX TABLE "+table); err != nil {
			return fmt.Errorf("reindexing %s: %w", table, err)
		}
	}
	return nil
}
