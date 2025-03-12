package datastore

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/central/cve/common"
	"github.com/stackrox/rox/central/cve/node/datastore/search"
	"github.com/stackrox/rox/central/cve/node/datastore/store"
	pgStore "github.com/stackrox/rox/central/cve/node/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/postgres"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchNodeCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.NodeCVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.NodeCVE, error)

	UpsertMany(ctx context.Context, cves []*storage.NodeCVE) error
	PruneNodeCVEs(ctx context.Context, ids []string) error

	// Suppress suppresses node vulnerabilities with provided cve names (not ids) for the duration provided.
	Suppress(ctx context.Context, start *time.Time, duration *time.Duration, cves ...string) error
	// Unsuppress unsuppresses node vulnerabilities with provided cve names (not ids).
	Unsuppress(ctx context.Context, cves ...string) error
	EnrichNodeWithSuppressedCVEs(node *storage.Node)
}

// New returns a new instance of a DataStore.
func New(storage store.Store, searcher search.Searcher, kf concurrency.KeyFence) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		searcher: searcher,

		cveSuppressionCache: make(common.CVESuppressionCache),
		keyFence:            kf,
	}
	if err := ds.buildSuppressedCache(); err != nil {
		return nil, err
	}
	return ds, nil
}

// GetTestPostgresDataStore provides a datastore connected to postgres for testing purposes.
func GetTestPostgresDataStore(_ testing.TB, pool postgres.DB) (DataStore, error) {
	dbstore := pgStore.New(pool)
	searcher := search.New(dbstore)
	return New(dbstore, searcher, concurrency.NewKeyFence())
}
