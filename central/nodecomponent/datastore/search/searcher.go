package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponent/datastore/index"
	pgStore "github.com/stackrox/rox/central/nodecomponent/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

// Searcher provides search functionality on existing image components.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchNodeComponents(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawNodeComponents(ctx context.Context, query *v1.Query) ([]*storage.NodeComponent, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage pgStore.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	scopedSafeSearcher := pkgPostgres.WithScoping(searcher)
	return sortfields.TransformSortFields(scopedSafeSearcher, schema.NodesSchema.OptionsMap)
}
