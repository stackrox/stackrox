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
	"github.com/stackrox/rox/scanner/datastore/postgres/migrations"
)

// MatcherMetadataStore represents a matcher metadata datastore.
//
//go:generate mockgen-wrapper
type MatcherMetadataStore interface {
	GetLastVulnerabilityUpdate(context.Context) (time.Time, error)
	SetLastVulnerabilityUpdate(context.Context, time.Time) error
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
