package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/complianceoperator/v2/benchmarks/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var _ Searcher = (*searcherImpl)(nil)

type searcherImpl struct {
	storage pgStore.Store
}

func (s searcherImpl) Search(ctx context.Context, query *v1.Query) ([]search.Result, error) {
	return s.storage.Search(ctx, query)
}

func (s searcherImpl) Count(ctx context.Context, query *v1.Query) (int, error) {
	return s.storage.Count(ctx, query)
}

func (s searcherImpl) SearchBenchmarks(ctx context.Context, query *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.storage.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	return s.resultsToSearchResults(ctx, results)
}

func (s searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	components, missingIndices, err := s.resultsToBenchmarks(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(components, results), nil
}

func convertMany(components []*storage.ComplianceOperatorBenchmarkV2, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(components))
	for i, sar := range components {
		outputResults[i] = convertOne(sar, &results[i])
	}
	return outputResults
}

func convertOne(benchmark *storage.ComplianceOperatorBenchmarkV2, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_COMPLIANCE_BENCHMARKS,
		Id:             benchmark.GetId(),
		Name:           benchmark.GetName(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (s searcherImpl) resultsToBenchmarks(ctx context.Context, results []search.Result) ([]*storage.ComplianceOperatorBenchmarkV2, []int, error) {
	return s.storage.GetMany(ctx, search.ResultsToIDs(results))
}
