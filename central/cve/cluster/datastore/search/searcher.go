package search

import (
	"context"

	"github.com/stackrox/rox/central/cve/cluster/datastore/index"
	"github.com/stackrox/rox/central/cve/cluster/datastore/store"
	"github.com/stackrox/rox/central/cve/edgefields"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
)

// Searcher provides search functionality on existing cves.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchClusterCVEs(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawClusterCVEs(ctx context.Context, query *v1.Query) ([]*storage.ClusterCVE, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	scopedSearcher := pkgPostgres.WithScoping(searcher)
	return edgefields.TransformFixableFields(scopedSearcher)
}
