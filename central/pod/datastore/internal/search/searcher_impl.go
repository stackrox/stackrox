package search

import (
	"context"

	"github.com/stackrox/rox/central/pod/mappings"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sortfields"
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
	searcher search.Searcher
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(podIndexer blevesearch.UnsafeSearcher) search.Searcher {
	filteredSearcher := podsSearchHelper.FilteredSearcher(podIndexer) // Make the UnsafeSearcher safe.
	transformedSortFieldSearcher := sortfields.TransformSortFields(filteredSearcher)
	paginatedSearcher := paginated.Paginated(transformedSortFieldSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
