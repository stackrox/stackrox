package search

import (
	"context"

	"github.com/stackrox/rox/central/blob/datastore/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage store.Store
}

func (s *searcherImpl) SearchIDs(ctx context.Context, q *v1.Query) ([]string, error) {
	results, err := s.storage.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	return ids, nil
}

func (s *searcherImpl) SearchMetadata(ctx context.Context, q *v1.Query) ([]*storage.Blob, error) {
	return s.storage.GetMetadataByQuery(ctx, q)
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.storage.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.storage.Count(ctx, q)
}
