package search

import (
	"context"

	"github.com/stackrox/rox/central/nodecomponentedge/index"
	"github.com/stackrox/rox/central/nodecomponentedge/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	sacHelper = sac.ForResource(resources.Node).MustCreatePgSearchHelper()
)

// Searcher provides search functionality on existing node component edges.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchEdges(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawEdges(ctx context.Context, query *v1.Query) ([]*storage.NodeComponentEdge, error)
}

// New returns a new instance of Searcher for the given storage and index.
func New(storage store.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage: storage,
		indexer: indexer,
		searcher: func() search.Searcher {
			if env.PostgresDatastoreEnabled.BooleanSetting() {
				return formatSearcherV2(indexer)
			}
			return formatSearcher(indexer)
		}(),
	}
}

func formatSearcherV2(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	scopedSafeSearcher := pkgPostgres.WithScoping(sacHelper.FilteredSearcher(unsafeSearcher))
	return sortfields.TransformSortFields(scopedSafeSearcher, schema.NodesSchema.OptionsMap)
}
