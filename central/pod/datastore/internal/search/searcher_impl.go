package search

import (
	"context"

	"github.com/stackrox/rox/central/pod/mappings"
	"github.com/stackrox/rox/central/pod/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/role/resources"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	podsSearchHelper = sac.ForResource(resources.Deployment).MustCreateSearchHelper(mappings.OptionsMap)

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

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

func (ds *searcherImpl) SearchRawPods(ctx context.Context, q *v1.Query) ([]*storage.Pod, error) {
	results, err := ds.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	ids := search.ResultsToIDs(results)
	pods, _, err := ds.storage.GetMany(ctx, ids)
	return pods, err
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(podIndexer blevesearch.UnsafeSearcher) search.Searcher {
	filteredSearcher := podsSearchHelper.FilteredSearcher(podIndexer) // Make the UnsafeSearcher safe.
	paginatedSearcher := paginated.Paginated(filteredSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
