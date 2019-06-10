package search

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/mappings"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/central/role/resources"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
)

var (
	indicatorSACSearchHelper = sac.ForResource(resources.Indicator).MustCreateSearchHelper(mappings.OptionsMap, sac.ClusterIDAndNamespaceFields)
)

// searcherImpl provides an intermediary implementation layer for ProcessStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

// SearchRawIndicators retrieves Policies from the indexer and storage
func (s *searcherImpl) SearchRawProcessIndicators(ctx context.Context, q *v1.Query) ([]*storage.ProcessIndicator, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}
	indicators := make([]*storage.ProcessIndicator, 0, len(results))
	for _, result := range results {
		indicator, exists, err := s.storage.GetProcessIndicator(result.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "retrieving indicator with id '%s'", result.ID)
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		indicators = append(indicators, indicator)
	}
	return indicators, nil
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return indicatorSACSearchHelper.Apply(s.indexer.Search)(ctx, q)
}
