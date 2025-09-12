package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/remind101/migrate"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/datastore/postgres/migrations"
)

// MatcherMetadataStore represents a matcher metadata datastore.
//
//go:generate mockgen-wrapper
type MatcherMetadataStore interface {
	// GetLastVulnerabilityUpdate returns a timestamp representing the last update timestamp of all vulnerability bundles.
	GetLastVulnerabilityUpdate(ctx context.Context) (time.Time, error)
	// GetLastVulnerabilityBundlesUpdate returns a timestamp representing the last update timestamp of given vuln bundle list.
	GetLastVulnerabilityBundlesUpdate(ctx context.Context, bundles []string) (map[string]time.Time, error)
	// SetLastVulnerabilityUpdate sets the last update timestamp of one vulnerability bundle.
	SetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) error
	// GetOrSetLastVulnerabilityUpdate get the last update timestamp of one vulnerability bundle,
	// or set it to the specified value if it does not exist.
	GetOrSetLastVulnerabilityUpdate(ctx context.Context, bundle string, lastUpdate time.Time) (time.Time, error)
	// GCVulnerabilityUpdates clean-up unknown and obsolete update timestamps.
	GCVulnerabilityUpdates(ctx context.Context, activeUpdaters []string, lastUpdate time.Time) error
}

type matcherMetadataStore struct {
	pool *pgxpool.Pool
}

// InitPostgresMatcherMetadataStore initializes a matcher metadata datastore.
func InitPostgresMatcherMetadataStore(_ context.Context, pool *pgxpool.Pool, doMigration bool) (MatcherMetadataStore, error) {
	if pool == nil {
		return nil, errors.New("pool must be non-nil")
	}

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer utils.IgnoreError(db.Close)

	if doMigration {
		migrator := migrate.NewPostgresMigrator(db)
		migrator.Table = migrations.MatcherMigrationTable
		err := migrator.Exec(migrate.Up, migrations.MatcherMigrations...)
		if err != nil {
			return nil, fmt.Errorf("failed to perform migrations: %w", err)
		}
	}

	return &matcherMetadataStore{
		pool: pool,
	}, nil
}
