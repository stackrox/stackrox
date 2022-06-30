package search

import (
	"context"

	"github.com/stackrox/rox/central/policycategory/index"
	categoryMapping "github.com/stackrox/rox/central/policycategory/index/mappings"
	"github.com/stackrox/rox/central/policycategory/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/search/paginated"
	"github.com/stackrox/rox/pkg/search/sortfields"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field: search.PolicyCategoryName.String(),
	}

	policySAC = sac.ForResource(resources.Policy)
)

// searcherImpl provides an intermediary implementation layer for AlertStorage.
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return nil, err
	}

	return s.searcher.Search(ctx, q)
}

func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if ok, err := policySAC.ReadAllowed(ctx); err != nil || !ok {
		return 0, err
	}

	return s.searcher.Count(ctx, q)
}

func (s searcherImpl) SearchRawCategories(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategory, error) {
	return s.searchCategories(ctx, q)
}

func (s *searcherImpl) searchCategories(ctx context.Context, q *v1.Query) ([]*storage.PolicyCategory, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	var categories []*storage.PolicyCategory
	for _, result := range results {
		category, exists, err := s.storage.Get(ctx, result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		categories = append(categories, category)
	}
	return categories, nil
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	safeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(unsafeSearcher)
	transformedSortFieldSearcher := sortfields.TransformSortFields(safeSearcher, categoryMapping.OptionsMap)
	paginatedSearcher := paginated.Paginated(transformedSortFieldSearcher)
	return paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
}
