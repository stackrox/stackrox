package search

import (
	"fmt"

	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
)

// searcherImpl provides an intermediary implementation layer for ProcessStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (s *searcherImpl) buildIndex() error {
	indicators, err := s.storage.GetProcessIndicators()
	if err != nil {
		return err
	}
	return s.indexer.AddProcessIndicators(indicators)
}

// SearchRawIndicators retrieves Policies from the indexer and storage
func (s *searcherImpl) SearchRawProcessIndicators(q *v1.Query) ([]*storage.ProcessIndicator, error) {
	results, err := s.indexer.Search(q)
	if err != nil {
		return nil, err
	}
	indicators := make([]*storage.ProcessIndicator, 0, len(results))
	for _, result := range results {
		indicator, exists, err := s.storage.GetProcessIndicator(result.ID)
		if err != nil {
			return nil, fmt.Errorf("retrieving indicator with id '%s': %s", result.ID, err)
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		indicators = append(indicators, indicator)
	}
	return indicators, nil
}
