package search

import (
	"github.com/stackrox/rox/central/processindicator/index"
	"github.com/stackrox/rox/central/processindicator/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
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
func (s *searcherImpl) SearchRawProcessIndicators(q *v1.Query) ([]*v1.ProcessIndicator, error) {
	indicators, _, err := s.searchIndicators(q)
	return indicators, err
}

// SearchIndicators retrieves SearchResults from the indexer and storage
func (s *searcherImpl) SearchProcessIndicators(q *v1.Query) ([]*v1.SearchResult, error) {
	indicators, results, err := s.searchIndicators(q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(indicators))
	for i, indicator := range indicators {
		protoResults = append(protoResults, convertIndicator(indicator, results[i]))
	}
	return protoResults, nil
}

func (s *searcherImpl) searchIndicators(q *v1.Query) ([]*v1.ProcessIndicator, []search.Result, error) {
	results, err := s.indexer.SearchProcessIndicators(q)
	if err != nil {
		return nil, nil, err
	}
	var indicators []*v1.ProcessIndicator
	var newResults []search.Result
	for _, result := range results {
		indicator, exists, err := s.storage.GetProcessIndicator(result.ID)
		if err != nil {
			return nil, nil, err
		}
		// The result may not exist if the object was deleted after the search
		if !exists {
			continue
		}
		indicators = append(indicators, indicator)
		newResults = append(newResults, result)
	}
	return indicators, newResults, nil
}

// convertIndicator returns proto search result from a indicator object and the internal search result
func convertIndicator(indicator *v1.ProcessIndicator, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_PROCESS_INDICATORS,
		Id:             indicator.GetId(),
		Name:           indicator.GetSignal().GetSignal().(*v1.Signal_ProcessSignal).ProcessSignal.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
