package search

import (
	"context"

	"github.com/stackrox/rox/central/reports/metadata/datastore/index"
	pgStore "github.com/stackrox/rox/central/reports/metadata/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var (
	defaultSortOption = &v1.QuerySortOption{
		Field:    search.ReportCompletionTime.String(),
		Reversed: true,
	}
)

type searcherImpl struct {
	storage  pgStore.Store
	indexer  index.Indexer
	searcher search.Searcher
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImpl) SearchReportMetadatas(ctx context.Context, q *v1.Query) ([]*storage.ReportMetadata, error) {
	reports, _, err := s.searchReportMetadatas(ctx, q)
	return reports, err
}

func (s *searcherImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	reports, results, err := s.searchReportMetadatas(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(reports))
	for i, report := range reports {
		protoResults = append(protoResults, convertOne(report, results[i]))
	}
	return protoResults, nil
}

func (s *searcherImpl) searchReportMetadatas(ctx context.Context, q *v1.Query) ([]*storage.ReportMetadata, []search.Result, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	reports, missingIndices, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}

	results = search.RemoveMissingResults(results, missingIndices)
	return reports, results, err
}

func convertOne(report *storage.ReportMetadata, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_REPORT_METADATA,
		Id:             report.GetReportId(),
		Name:           report.GetReportId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
