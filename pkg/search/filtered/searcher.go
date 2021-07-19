package filtered

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// Filter represents a process of converting from one id-space to another.
type Filter interface {
	// Apply applies the filter to the elements. The return value is a slice of the indices that passed the filter,
	// alternatively a bool indicating that no elements should be removed by the filter. The error return value
	// always needs to be checked first.
	Apply(ctx context.Context, from ...string) ([]int, bool, error)
}

// UnsafeSearcher generates a Searcher from an UnsafeSearcher by filtering its outputs with the input filter.
func UnsafeSearcher(searcher blevesearch.UnsafeSearcher, filter Filter) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			results, err := searcher.Search(q)
			if err != nil {
				return results, err
			}

			if err := ApplySACFilterToSearchResults(ctx, &results, filter); err != nil {
				return nil, err
			}
			return results, nil
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			if filter == nil {
				return searcher.Count(q)
			}
			result, err := searcher.Search(q)
			if err != nil {
				return 0, err
			}
			err = ApplySACFilterToSearchResults(ctx, &result, filter)
			if err != nil {
				return 0, err
			}
			return len(result), nil
		},
	}
}

// Searcher returns a new searcher based on the filtered output from the input Searcher.
func Searcher(searcher search.Searcher, filter Filter) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			results, err := searcher.Search(ctx, q)
			if err != nil {
				return results, err
			}

			if err := ApplySACFilterToSearchResults(ctx, &results, filter); err != nil {
				return nil, err
			}

			return results, nil
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			if filter == nil {
				return searcher.Count(ctx, q)
			}
			// If we have SAC filters configured, we count search results.
			results, err := searcher.Search(ctx, q)
			if err != nil {
				return 0, err
			}
			err = ApplySACFilterToSearchResults(ctx, &results, filter)
			if err != nil {
				return 0, err
			}
			return len(results), nil
		},
	}
}

// ApplySACFilter filters ids with a SAC filter.
func ApplySACFilter(ctx context.Context, ids []string, filter Filter) ([]string, error) {
	filteredIndices, all, err := filter.Apply(ctx, ids...)
	if err != nil {
		return nil, err
	}
	if all || len(filteredIndices) == len(ids) {
		return ids, nil
	}
	return sliceutils.StringSelect(ids, filteredIndices...), nil
}

// ApplySACFilterToSearchResults filters search results with a SAC filter.
func ApplySACFilterToSearchResults(ctx context.Context, resultsInOut *[]search.Result, filter Filter) error {
	results := *resultsInOut

	ids := search.ResultsToIDs(results)
	filteredIndices, all, err := filter.Apply(ctx, ids...)
	if err != nil {
		return err
	}
	if all || len(filteredIndices) == len(results) {
		return nil
	}
	filteredResults := results[:0]
	for _, idx := range filteredIndices {
		filteredResults = append(filteredResults, results[idx])
	}
	*resultsInOut = filteredResults
	return nil
}
