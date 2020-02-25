package datastore

import (
	"context"

	"github.com/stackrox/rox/central/cve/converter"
	"github.com/stackrox/rox/central/cve/index"
	"github.com/stackrox/rox/central/cve/search"
	"github.com/stackrox/rox/central/cve/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	searchPkg "github.com/stackrox/rox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//go:generate mockgen-wrapper DataStore
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.CVE, bool, error)
	Count(ctx context.Context) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.CVE, error)

	Suppress(ctx context.Context, ids ...string) error
	Unsuppress(ctx context.Context, ids ...string) error

	Upsert(ctx context.Context, cves ...*storage.CVE) error
	UpsertClusterCVEs(ctx context.Context, cves ...converter.ClusterCVEParts) error
	Delete(ctx context.Context, ids ...string) error
}

// New returns a new instance of a DataStore.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,
	}
	return ds, nil
}
