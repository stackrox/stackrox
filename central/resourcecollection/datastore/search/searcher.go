package search

import (
	"context"

	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	"github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/postgres/schema"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	pkgPostgres "github.com/stackrox/rox/pkg/search/scoped/postgres"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	sacHelper = sac.ForResource(resources.WorkflowAdministration).MustCreatePgSearchHelper()
)

// Searcher provides search functionality on existing Collections.
//
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchResults(ctx context.Context, query *v1.Query) ([]*v1.SearchResult, error)
	SearchCollections(context.Context, *v1.Query) ([]*storage.ResourceCollection, error)
}

// New returns a new instance of Searcher for the given storage and indexer.
func New(storage postgres.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: formatSearcherV2(indexer),
	}
}

func formatSearcherV2(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	scopedSafeSearcher := pkgPostgres.WithScoping(sacHelper.FilteredSearcher(unsafeSearcher))
	return sortfields.TransformSortFields(scopedSafeSearcher, schema.CollectionsSchema.OptionsMap)
}
