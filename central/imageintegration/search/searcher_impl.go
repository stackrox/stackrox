package search

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/index"
	imageIntegrationMapping "github.com/stackrox/rox/central/imageintegration/index/mappings"
	"github.com/stackrox/rox/central/imageintegration/store"
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
		Field:    search.ClusterID.String(),
		Reversed: false,
	}
	imageIntegrationSAC = sac.ForResource(resources.ImageIntegration)
)

// searcherImpl provides an intermediary implementation layer for image integration.
type searcherImpl struct {
	storage  store.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (ds *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return ds.searcher.Search(ctx, q)
}

func (ds *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return ds.searcher.Count(ctx, q)
}

// Format the search functionality of the indexer to be filtered (for sac) and paginated.
func formatSearcher(unsafeSearcher blevesearch.UnsafeSearcher) search.Searcher {
	safeSearcher := blevesearch.WrapUnsafeSearcherAsSearcher(unsafeSearcher)
	transformedSortFieldSearcher := sortfields.TransformSortFields(safeSearcher, imageIntegrationMapping.OptionsMap)
	paginatedSearcher := paginated.Paginated(transformedSortFieldSearcher)
	return paginated.WithDefaultSortOption(paginatedSearcher, defaultSortOption)
}

// Search retrieves SearchResults from the indexer and storage
func (ds *searcherImpl) SearchImageIntegrations(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := ds.searcher.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	var imageIntegrationList []*storage.ImageIntegration
	for _, result := range results {
		singleImageIntegration, exists, err := ds.storage.Get(ctx, result.ID)
		if err != nil {
			return nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		imageIntegrationList = append(imageIntegrationList, singleImageIntegration)
	}

	protoResults := make([]*v1.SearchResult, 0, len(imageIntegrationList))
	for i, imageIntegration := range imageIntegrationList {
		protoResults = append(protoResults, convertImageIntegration(imageIntegration, results[i]))
	}
	return protoResults, nil
}

// convertPolicy returns proto search result from a policy object and the internal search result
func convertImageIntegration(imageIntegration *storage.ImageIntegration, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGE_INTEGRATIONS,
		Id:             imageIntegration.GetId(),
		Name:           imageIntegration.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
