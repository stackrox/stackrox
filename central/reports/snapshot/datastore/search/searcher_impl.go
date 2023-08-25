package search

import (
	"context"

	pgStore "github.com/stackrox/rox/central/reports/snapshot/datastore/store/postgres"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

type searcherImpl struct {
	storage  pgStore.Store
	searcher search.Searcher
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.searcher.Search(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImpl) SearchReportSnapshots(ctx context.Context, q *v1.Query) ([]*storage.ReportSnapshot, error) {
	snaps, _, err := s.searchReportSnapshots(ctx, q)
	return snaps, err
}

func (s *searcherImpl) SearchResults(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	snaps, results, err := s.searchReportSnapshots(ctx, q)
	if err != nil {
		return nil, err
	}
	protoResults := make([]*v1.SearchResult, 0, len(snaps))
	for i, snap := range snaps {
		protoResults = append(protoResults, convertOne(snap, results[i]))
	}
	return protoResults, nil
}

func (s *searcherImpl) searchReportSnapshots(ctx context.Context, q *v1.Query) ([]*storage.ReportSnapshot, []search.Result, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, nil, err
	}

	snaps, missingIndices, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, nil, err
	}

	results = search.RemoveMissingResults(results, missingIndices)
	return snaps, results, err
}

func convertOne(report *storage.ReportSnapshot, result search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_REPORT_SNAPSHOT,
		Id:             report.GetReportId(),
		Name:           report.GetReportId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}
