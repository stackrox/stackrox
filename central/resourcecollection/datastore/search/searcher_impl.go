package search

import (
	"context"

	"github.com/stackrox/rox/central/resourcecollection/datastore/index"
	pgStore "github.com/stackrox/rox/central/resourcecollection/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  pgStore.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImpl) SearchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, error) {
	clusters, _, err := s.searchCollections(ctx, q)
	return clusters, err
}

func (s *searcherImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	collections, results, err := s.searchCollections(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(collections))
	for i, collection := range collections {
		protoResults = append(protoResults, convertOne(collection, results[i]))
	}
	return protoResults, nil
}

func (s *searcherImpl) searchCollections(ctx context.Context, q *v1.Query) ([]*storage.ResourceCollection, []search.Result, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	collections, missingIndices, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}

	results = search.RemoveMissingResults(results, missingIndices)
	return collections, results, err
}

func convertOne(collection *storage.ResourceCollection, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_COLLECTIONS,
		Id:             collection.GetId(),
		Name:           collection.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
