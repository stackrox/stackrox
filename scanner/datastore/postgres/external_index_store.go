package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pkg/errors"
	"github.com/quay/claircore"
	"github.com/stackrox/rox/pkg/utils"
)

//go:generate mockgen-wrapper
type ExternalIndexStore interface {
	StoreIndexReport(ctx context.Context, hashID string, clusterName string, indexReport *claircore.IndexReport) error
	StoreIndexReportWithExpiration(ctx context.Context, hashID string, clusterName string, indexReport *claircore.IndexReport, expiration time.Time) error
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

	return &externalIndexStore{
		pool: pool,
	}, nil
}
