package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/remind101/migrate"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/matcher/metadata/migrations"
)

// MetadataStore represents a matcher metadata datastore.
//
//go:generate mockgen-wrapper
type MetadataStore interface {
	GetLastVulnerabilityUpdate(context.Context) (time.Time, error)
	SetLastVulnerabilityUpdate(context.Context, time.Time) error
}

type metadataStore struct {
	pool *pgxpool.Pool
}

// InitPostgresMetadataStore initializes a matcher metadata datastore.
func InitPostgresMetadataStore(_ context.Context, pool *pgxpool.Pool, doMigration bool) (MetadataStore, error) {
	if pool == nil {
		return nil, errors.New("pool must be non-nil")
	}

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer utils.IgnoreError(db.Close)

	if doMigration {
		migrator := migrate.NewPostgresMigrator(db)
		migrator.Table = migrations.MigrationTable
		err := migrator.Exec(migrate.Up, migrations.Migrations...)
		if err != nil {
			return nil, fmt.Errorf("failed to perform migrations: %w", err)
		}
	}

	return &metadataStore{
		pool: pool,
	}, nil
}
