package search

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/store"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/debug"
)

// searcherImpl provides an intermediary implementation layer for ProcessStorage.
type searcherImpl struct {
	storage store.Store
	indexer index.Indexer
}

func (s *searcherImpl) buildIndex() error {
	defer debug.FreeOSMemory()
	log.Info("[STARTUP] Indexing process indicators")
	indicators, err := s.storage.GetProcessIndicators()
	if err != nil {
		return err
	}
	if err := s.indexer.AddProcessIndicators(indicators); err != nil {
		return err
	}
	log.Info("[STARTUP] Successfully indexed process indicators")
	return nil
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
