package search

import (
	"context"

	"github.com/stackrox/stackrox/central/nodecomponent/datastore/index"
	"github.com/stackrox/stackrox/central/nodecomponent/datastore/store/postgres"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/blevesearch"
	pkgPostgres "github.com/stackrox/stackrox/pkg/search/scoped/postgres"
)

// Searcher provides search functionality on existing image components.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchNodeComponents(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawNodeComponents(ctx context.Context, query *v1.Query) ([]*storage.NodeComponent, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage postgres.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: pkgPostgres.WithScoping(blevesearch.WrapUnsafeSearcherAsSearcher(indexer)),
	}
}
