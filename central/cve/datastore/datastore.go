package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/central/cve/common"
	"github.com/stackrox/stackrox/central/cve/index"
	"github.com/stackrox/stackrox/central/cve/search"
	"github.com/stackrox/stackrox/central/cve/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/dackbox/graph"
	"github.com/stackrox/stackrox/pkg/dackbox/utils/queue"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.CVE, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.CVE, error)

	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, ids ...string) error
	Unsuppress(ctx context.Context, ids ...string) error
	EnrichImageWithSuppressedCVEs(image *storage.Image)
	EnrichNodeWithSuppressedCVEs(node *storage.Node)

	Delete(ctx context.Context, ids ...string) error
}

// New returns a new instance of a DataStore.
func New(graphProvider graph.Provider, indexQ queue.WaitableQueue, storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:       storage,
		indexer:       indexer,
		searcher:      searcher,
		graphProvider: graphProvider,
		indexQ:        indexQ,

		cveSuppressionCache: make(common.CVESuppressionCache),
	}
	if err := ds.buildSuppressedCache(); err != nil {
		return nil, err
	}
	return ds, nil
}
