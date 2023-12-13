package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/remind101/migrate"
	"github.com/stackrox/rox/scanner/matcher/metadata/migrations"
)

type MetadataStore struct {
	pool *pgxpool.Pool
}

// InitPostgresMetadataStore initializes a matcher metadata datastore.
func InitPostgresMetadataStore(_ context.Context, pool *pgxpool.Pool, doMigration bool) (*MetadataStore, error) {
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()

	if doMigration {
		migrator := migrate.NewPostgresMigrator(db)
		migrator.Table = migrations.MigrationTable
		err := migrator.Exec(migrate.Up, migrations.Migrations...)
		if err != nil {
			return nil, fmt.Errorf("failed to perform migrations: %w", err)
		}
	}

	store := NewMetadataStore(pool)
	return store, nil
}

// NewMetadataStore returns a Store using the passed-in pool.
func NewMetadataStore(pool *pgxpool.Pool) *MetadataStore {
	return &MetadataStore{
		pool: pool,
	}
}
