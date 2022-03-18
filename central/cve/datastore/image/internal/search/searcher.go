package search

import (
	"context"

	"github.com/stackrox/rox/central/cve/datastore/image/internal/store/postgres"
	"github.com/stackrox/rox/central/cve/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
)

// Searcher provides search functionality on existing cves.
//go:generate mockgen-wrapper
type Searcher interface {
	Search(ctx context.Context, query *v1.Query) ([]search.Result, error)
	Count(ctx context.Context, query *v1.Query) (int, error)
	SearchCVEs(context.Context, *v1.Query) ([]*v1.SearchResult, error)
	SearchRawCVEs(ctx context.Context, query *v1.Query) ([]*storage.CVE, error)
}

type searcherImpl struct {
	storage  postgres.Store
	indexer  index.Indexer
	searcher search.Searcher
}

// New returns a new instance of searcher.
func New(storage postgres.Store, indexer index.Indexer) Searcher {
	return &searcherImpl{
		storage:  storage,
		indexer:  indexer,
		searcher: blevesearch.WrapUnsafeSearcherAsSearcher(indexer),
	}
}

func (s *searcherImpl) SearchCVEs(ctx context.Context, q *v1.Query) ([]*v1.SearchResult, error) {
	results, err := s.getSearchResults(ctx, q)
	if err != nil {
		return nil, err
	}
	return s.resultsToSearchResults(ctx, results)
}

func (s *searcherImpl) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return s.getSearchResults(ctx, q)
}

// Count returns the number of search results from the query
func (s *searcherImpl) Count(ctx context.Context, q *v1.Query) (int, error) {
	return s.getCount(ctx, q)
}

func (s *searcherImpl) SearchRawCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
	return s.searchCVEs(ctx, q)
}

func (s *searcherImpl) getSearchResults(ctx context.Context, q *v1.Query) (res []search.Result, err error) {
	return s.searcher.Search(ctx, q)
}

func (s *searcherImpl) getCount(ctx context.Context, q *v1.Query) (count int, err error) {
	return s.searcher.Count(ctx, q)
}

func (s *searcherImpl) resultsToCVEs(ctx context.Context, results []search.Result) ([]*storage.CVE, []int, error) {
	return s.storage.GetMany(ctx, search.ResultsToIDs(results))
}

func (s *searcherImpl) resultsToSearchResults(ctx context.Context, results []search.Result) ([]*v1.SearchResult, error) {
	cves, missingIndices, err := s.resultsToCVEs(ctx, results)
	if err != nil {
		return nil, err
	}
	results = search.RemoveMissingResults(results, missingIndices)
	return convertMany(cves, results), nil
}

func convertMany(cves []*storage.CVE, results []search.Result) []*v1.SearchResult {
	outputResults := make([]*v1.SearchResult, len(cves))
	for index, sar := range cves {
		outputResults[index] = convertOne(sar, &results[index])
	}
	return outputResults
}

func convertOne(cve *storage.CVE, result *search.Result) *v1.SearchResult {
	return &v1.SearchResult{
		Category:       v1.SearchCategory_IMAGE_VULNERABILITIES,
		Id:             cve.GetId(),
		Name:           cve.GetId(),
		FieldToMatches: search.GetProtoMatchesMap(result.Matches),
		Score:          result.Score,
	}
}

func (s *searcherImpl) searchCVEs(ctx context.Context, q *v1.Query) ([]*storage.CVE, error) {
	results, err := s.Search(ctx, q)
	if err != nil {
		return nil, err
	}

	cves, _, err := s.storage.GetMany(ctx, search.ResultsToIDs(results))
	if err != nil {
		return nil, err
	}
	return cves, nil
}
