package search

import (
	"context"

	"github.com/stackrox/stackrox/central/processindicator/index"
	"github.com/stackrox/stackrox/central/processindicator/store"
	"github.com/stackrox/stackrox/central/role/resources"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/features"
	"github.com/stackrox/stackrox/pkg/sac"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/options/processindicators"
)

var (
	indicatorSACSearchHelper         = sac.ForResource(resources.Indicator).MustCreateSearchHelper(processindicators.OptionsMap)
	indicatorSACPostgresSearchHelper = sac.ForResource(resources.Indicator).MustCreatePgSearchHelper()
)

// searcherImpl provides an intermediary implementation layer for ProcessStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchRawProcessIndicators retrieves Policies from the indexer and storage
func (s *searcherImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	processes, _, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	return processes, err
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	if features.PostgresDatastore.Enabled() {
		return indicatorSACPostgresSearchHelper.Apply(s.indexer.Search)(ctx, q)
	}
	return indicatorSACSearchHelper.Apply(s.indexer.Search)(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if features.PostgresDatastore.Enabled() {
		return indicatorSACPostgresSearchHelper.ApplyCount(s.indexer.Count)(ctx, q)
	}
	return indicatorSACSearchHelper.ApplyCount(s.indexer.Count)(ctx, q)
}
