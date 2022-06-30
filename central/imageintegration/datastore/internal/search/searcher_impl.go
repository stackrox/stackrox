package search

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.ClusterID.String(),
		Reversed: false,
	}
)

// searcherImpl provides an intermediary implementation layer for PodStorage.
type searcherImpl struct {
	storage  store.Store
	searcher search.Searcher
}

func (ds searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	res, err := ds.searcher.Search(ctx, q)
	if err != nil {
		log.Error(">>>>Image integration search_impl Search error")
		log.Error(err.Error())
		return nil, err
	}
	return res, nil
}

func (ds searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	safeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(unsafeSearcher) // Make the UnsafeSearcher safe.
	paginatedSearcher := paginated.Paginated(safeSearcher)
	defaultSortedSearcher := paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
	return defaultSortedSearcher
}
