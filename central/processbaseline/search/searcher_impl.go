package search

import (
	"context"

	"github.com/stackrox/rox/central/processbaseline/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage store.Store
}

func (s *searcherImpl) SearchRawProcessBaselines(ctx context.Context, q *v1.Query) ([]*storage.ProcessBaseline, error) {
	results, err := s.storage.Search(ctx, q)
	if err != nil || len(results) == 0 {
		return nil, err
	}
	ids := search.ResultsToIDs(results)
	baselines, _, err := s.storage.GetMany(ctx, ids)
	if err != nil {
		return nil, err
	}
	return baselines, nil
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.storage.Search(ctx, q)
}
