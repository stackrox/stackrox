package filtered

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/blevesearch"
	"github.com/stackrox/rox/pkg/set"
)

// Filter represents a process of converting from one id-space to another.
type Filter interface {
	Apply(ctx context.Context, from ...string) ([]string, error)
}

// UnsafeSearcher generates a Searcher from an UnsafeSearcher by filtering its outputs with the input filter.
func UnsafeSearcher(searcher blevesearch.UnsafeSearcher, filter Filter) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			results, err := searcher.Search(q)
			if err != nil {
				return results, err
			}

			allFiltered, err := ApplySACFilter(ctx, search.ResultsToIDs(results), filter)
			if err != nil {
				return nil, err
			}

			filteredResults := results[:0]
			filteredSet := set.NewStringSet(allFiltered...)
			for _, result := range results {
				if filteredSet.Contains(result.ID) {
					filteredResults = append(filteredResults, result)
				}
			}
			return filteredResults, nil
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return searcher.Count(q)
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

			allFiltered, err := ApplySACFilter(ctx, search.ResultsToIDs(results), filter)
			if err != nil {
				return nil, err
			}

			filteredResults := results[:0]
			filteredSet := set.NewStringSet(allFiltered...)
			for _, result := range results {
				if filteredSet.Contains(result.ID) {
					filteredResults = append(filteredResults, result)
				}
			}
			return filteredResults, nil
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

			filtered, err := ApplySACFilter(ctx, search.ResultsToIDs(results), filter)
			if err != nil {
				return 0, err
			}
			return len(filtered), nil
		},
	}
}

// ApplySACFilter filters ids with sac filters
func ApplySACFilter(ctx context.Context, ids []string, filter Filter) ([]string, error) {
	filtered, err := filter.Apply(ctx, ids...)
	if err != nil {
		return nil, err
	}
	return filtered, nil
}
