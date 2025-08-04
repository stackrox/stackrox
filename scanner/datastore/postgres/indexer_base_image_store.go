package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/remind101/migrate"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/baseimage"
	"github.com/stackrox/rox/scanner/datastore/postgres/migrations"
)

// IndexerBaseImageStore represents an indexer base image datastore.
//
//go:generate mockgen-wrapper
type IndexerBaseImageStore interface {
	// AddBaseImage adds a new base image and its associated layers to the database.
	AddBaseImage(ctx context.Context, baseImage baseimage.AddBaseImageInput) error

	GetBaseImageCandidates(ctx context.Context, digest string) (map[string][]string, error)
}

type indexerBaseImageStore struct {
	pool *pgxpool.Pool
}

// InitPostgresIndexerBaseImageStore initializes an indexer base image datastore.
func InitPostgresIndexerBaseImageStore(ctx context.Context, pool *pgxpool.Pool, doMigration bool) (IndexerBaseImageStore, error) {
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

	return &indexerBaseImageStore{
		pool: pool,
	}, nil
}
