package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponentedge/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/search"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

// Searcher provides search functionality on existing node component edges.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchEdges(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, query *v1.Query) ([]*storage.NodeComponentEdge, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store) Searcher {
	return &searcherImpl{
		storage:  storage,
		searcher: formatSearcherV2(storage),
	}
}

func formatSearcherV2(searcher search.Searcher) search.Searcher {
	scopedSafeSearcher := pkgPostgres.WithScoping(searcher)
	return sortfields.TransformSortFields(scopedSafeSearcher, schema.NodesSchema.OptionsMap)
}
