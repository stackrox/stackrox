package search

import (
	"context"

	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/search"
)

var (
	deploymentExtensionSACPostgresSearchHelper = sac.ForResource(resources.DeploymentExtension).MustCreatePgSearchHelper()
)

// searcherImpl provides an intermediary implementation layer for ProcessStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchRawProcessIndicators retrieves Policies from the indexer and storage
func (s *searcherImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	return s.storage.GetByQuery(ctx, q)
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	// return deploymentExtensionSACPostgresSearchHelper.FilteredSearcher(s.indexer).Search(ctx, q)
	return s.indexer.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	// return deploymentExtensionSACPostgresSearchHelper.FilteredSearcher(s.indexer).Count(ctx, q)
	return s.indexer.Count(ctx, q)
}
