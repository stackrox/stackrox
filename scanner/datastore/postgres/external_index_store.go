package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/remind101/migrate"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/scanner/datastore/postgres/migrations"
)

type ExternalIndexStore interface {
	StoreIndexReport(ctx context.Context, hashID string, indexReport *claircore.IndexReport, expiration time.Time) error
	GCIndexReports(ctx context.Context, expiration time.Time, opts ...ReindexGCOption) ([]string, error)
}

type externalIndexStore struct {
	pool *pgxpool.Pool
}

// InitPostgresExternalIndexStore initializes an external index report datastore.
func InitPostgresExternalIndexStore(_ context.Context, pool *pgxpool.Pool) (ExternalIndexStore, error) {
	if pool == nil {
		return nil, errors.New("pool must be non-nil")
	}

	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer utils.IgnoreError(db.Close)

	migrator := migrate.NewPostgresMigrator(db)
	migrator.Table = migrations.IndexerMigrationTable
	err := migrator.Exec(migrate.Up, migrations.IndexerMigrations...)
	if err != nil {
		return nil, fmt.Errorf("failed to perform migrations: %w", err)
	}

	return &externalIndexStore{
		pool: pool,
	}, nil
}
