package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/cel-go/common/stdlib"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/baseimage"
	"github.com/stackrox/rox/scanner/datastore/postgres/migrations"
)

// IndexeBaseImageStore represents an indexer base image datastore.
//
//go:generate mockgen-wrapper
type IndexerBaseImageStore interface {
	// AddBaseImage adds a new base image and its associated layers to the database.
	AddBaseImage(ctx context.Context, input baseimage.AddBaseImageInput) error
}

type indexerBaseImageStore struct {
	pool *pgxpool.Pool
}

// InitPostgresIndexerBaseImageStore initializes a indexer base image datastore.
func InitPostgresIndexerBaseImageStore(_ context.Context, pool *pgxpool.Pool, doMigration bool) (IndexerBaseImageStore, error) {
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

	return &indexerBaseImageStore{
		pool: pool
	}, nil
}
