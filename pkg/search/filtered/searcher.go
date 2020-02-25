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
func UnsafeSearcher(searcher blevesearch.UnsafeSearcher, filters ...Filter) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		results, err := searcher.Search(q)
		if err != nil {
			return results, err
		}

		allFiltered, err := ApplySACFilters(ctx, search.ResultsToIDs(results), filters...)
		if err != nil {
			return results, err
		}

		filteredResults := results[:0]
		filteredSet := set.NewStringSet(allFiltered...)
		for _, result := range results {
			if filteredSet.Contains(result.ID) {
				filteredResults = append(filteredResults, result)
			}
		}
		return filteredResults, nil
	})
}

// Searcher returns a new searcher based on the filtered output from the input Searcher.
func Searcher(searcher search.Searcher, filters ...Filter) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		results, err := searcher.Search(ctx, q)
		if err != nil {
			return results, err
		}

		allFiltered, err := ApplySACFilters(ctx, search.ResultsToIDs(results), filters...)
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
	})
}

// ApplySACFilters filters ids with sac filters
func ApplySACFilters(ctx context.Context, ids []string, filters ...Filter) ([]string, error) {
	var allFiltered []string
	for _, filter := range filters {
		filtered, err := filter.Apply(ctx, ids...)
		if err != nil {
			return ids, err
		} else if len(filtered) == 0 {
			continue
		}

		allFiltered = append(allFiltered, filtered...)
		// evaluate unfiltered ids on other sac filters
		ids = set.NewStringSet(ids...).Difference(set.NewStringSet(filtered...)).AsSlice()
	}
	return allFiltered, nil
}
