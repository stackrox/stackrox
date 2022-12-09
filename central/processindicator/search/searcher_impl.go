package search

import (
	"context"

	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/options/processindicators"
)

var (
	deploymentExtensionSACSearchHelper         = sac.ForResource(resources.DeploymentExtension).MustCreateSearchHelper(processindicators.OptionsMap)
	deploymentExtensionSACPostgresSearchHelper = sac.ForResource(resources.DeploymentExtension).MustCreatePgSearchHelper()
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
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return deploymentExtensionSACPostgresSearchHelper.FilteredSearcher(s.indexer).Search(ctx, q)
	}
	return deploymentExtensionSACSearchHelper.FilteredSearcher(s.indexer).Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	if env.PostgresDatastoreEnabled.BooleanSetting() {
		return deploymentExtensionSACPostgresSearchHelper.FilteredSearcher(s.indexer).Count(ctx, q)
	}
	return deploymentExtensionSACSearchHelper.FilteredSearcher(s.indexer).Count(ctx, q)
}
