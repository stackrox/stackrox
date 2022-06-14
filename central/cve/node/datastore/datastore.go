package datastore

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/central/cve/common"
	"github.com/stackrox/stackrox/central/cve/node/datastore/internal/index"
	"github.com/stackrox/stackrox/central/cve/node/datastore/internal/search"
	"github.com/stackrox/stackrox/central/cve/node/datastore/internal/store"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	searchPkg "github.com/stackrox/stackrox/pkg/search"
)

// DataStore is an intermediary to CVE storage.
//go:generate mockgen-wrapper
type DataStore interface {
	Search(ctx context.Context, q *v1.Query) ([]searchPkg.Result, error)
	SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.NodeCVE, error)

	Exists(ctx context.Context, id string) (bool, error)
	Get(ctx context.Context, id string) (*storage.NodeCVE, bool, error)
	Count(ctx context.Context, q *v1.Query) (int, error)
	GetBatch(ctx context.Context, id []string) ([]*storage.NodeCVE, error)

	// Suppress suppresses node vulnerabilities with provided cve names (not ids) for the duration provided.
	Suppress(ctx context.Context, start *types.Timestamp, duration *types.Duration, cves ...string) error
	// Unsuppress unsuppresses node vulnerabilities with provided cve names (not ids).
	Unsuppress(ctx context.Context, cves ...string) error
	EnrichNodeWithSuppressedCVEs(node *storage.Node)
}

// New returns a new instance of a DataStore.
func New(storage store.Store, indexer index.Indexer, searcher search.Searcher) (DataStore, error) {
	ds := &datastoreImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: searcher,

		cveSuppressionCache: make(common.CVESuppressionCache),
	}
	if err := ds.buildSuppressedCache(); err != nil {
		return nil, err
	}
	return ds, nil
}
