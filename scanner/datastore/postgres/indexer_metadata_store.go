package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/quay/claircore/indexer"
	"github.com/remind101/migrate"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/datastore/postgres/migrations"
)

// IndexerMetadataStore represents an indexer metadata datastore.
//
//go:generate mockgen-wrapper
type IndexerMetadataStore interface {
	// MigrateManifests populates the IndexerMetadataStore with manifests in ClairCore's manifest table
	// which do not yet exist in the manifest_metadata table and sets the expiration to the given expiration.
	//
	// If not already, the expiration time will be converted to UTC timezone.
	MigrateManifests(ctx context.Context, expiration time.Time) ([]string, error)
	// StoreManifest stores the given manifest ID into the manifest_metadata table to be deleted by
	// GCManifests after expiration passes.
	//
	// If not already, the expiration time will be converted to UTC timezone.
	StoreManifest(ctx context.Context, manifestID string, expiration time.Time) error
	// ManifestExists returns if the given manifest exists in the table.
	ManifestExists(ctx context.Context, manifestID string) (bool, error)
	// GCManifests deletes manifests from the manifest_metadata table with timestamps older than expiration (converted to UTC)
	// and returns their respective IDs.
	GCManifests(ctx context.Context, expiration time.Time, opts ...GCManifestsOption) ([]string, error)
}

type indexerMetadataStore struct {
	pool *pgxpool.Pool

	store indexer.Store
}

// IndexerMetadataStoreOpts defines options for creating an IndexerMetadataStore.
type IndexerMetadataStoreOpts struct {
	// IndexerStore represents the indexer.Store to query when MigrateManifests and GCManifests are called.
	// If undefined, then MigrateManifests will fail.
	IndexerStore indexer.Store
}

// GCManifestsOption is a configuration option for the GCManifests method.
type GCManifestsOption func(o *gcManifestsOpts)

type gcManifestsOpts struct {
	gcThrottle int
}

// WithGCThrottle sets the maximum number of manifests to GC.
// Default: 100
func WithGCThrottle(gcThrottle int) GCManifestsOption {
	return func(o *gcManifestsOpts) {
		o.gcThrottle = gcThrottle
	}
}

func makeGCManifestsOpts(opts []GCManifestsOption) gcManifestsOpts {
	var o gcManifestsOpts
	for _, opt := range opts {
		opt(&o)
	}

	if o.gcThrottle == 0 {
		o.gcThrottle = 100
	}

	return o
}

// InitPostgresIndexerMetadataStore initializes an indexer metadata datastore.
func InitPostgresIndexerMetadataStore(_ context.Context, pool *pgxpool.Pool, doMigration bool, opts IndexerMetadataStoreOpts) (IndexerMetadataStore, error) {
	if pool == nil {
		return nil, errors.New("pool must be non-nil")
	}

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer utils.IgnoreError(db.Close)

	if doMigration {
		migrator := migrate.NewPostgresMigrator(db)
		migrator.Table = migrations.IndexerMigrationTable
		err := migrator.Exec(migrate.Up, migrations.IndexerMigrations...)
		if err != nil {
			return nil, fmt.Errorf("failed to perform migrations: %w", err)
		}
	}

	return &indexerMetadataStore{
		pool: pool,

		store: opts.IndexerStore,
	}, nil
}
