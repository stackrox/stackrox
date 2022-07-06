package search

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/mappings"
	"github.com/stackrox/rox/central/imageintegration/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	imageintegrationsSACSearchHelper         = sac.ForResource(resources.ImageIntegration).MustCreateSearchHelper(mappings.OptionsMap)
	imageintegrationsSACPostgresSearchHelper = sac.ForResource(resources.ImageIntegration).MustCreatePgSearchHelper()

	defaultSortOption = &v1.QuerySortOption{
		Field:    search.DeploymentID.String(),
		Reversed: false,
	}
)

// searcherImpl provides an intermediary implementation layer for PodStorage.
type searcherImpl struct {
	storage  store.Store
	searcher search.Searcher
}

func (ds searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

//func (ds searcherImpl) SearchImageIntegration(ctx context.Context, q *v1.Query) ([]*storage.ImageIntegration, error) {
//	results, err := ds.Search(ctx, q)
//	if err != nil {
//		return nil, err
//	}
//
//	ids := search.ResultsToIDs(results)
//	iis, _, err := ds.storage.GetMany(ctx, ids)
//	return iis, err
//}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(indexer blevesearch.UnsafeSearcher) search.Searcher {
	var filteredSearcher search.Searcher
	if features.PostgresDatastore.Enabled() {
		filteredSearcher = imageintegrationsSACPostgresSearchHelper.FilteredSearcher(indexer) // Make the UnsafeSearcher safe.
	} else {
		filteredSearcher = imageintegrationsSACSearchHelper.FilteredSearcher(indexer) // Make the UnsafeSearcher safe.
	}
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
