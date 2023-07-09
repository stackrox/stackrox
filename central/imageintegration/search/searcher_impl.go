package search

import (
	"context"

	"github.com/stackrox/rox/central/imageintegration/index"
	"github.com/stackrox/rox/central/imageintegration/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
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

// SearchImageIntegrations retrieves SearchResults from the indexer and storage
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
