package derivedfields

import (
	"context"
	"sort"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/derivedfields/counter"
	"github.com/stackrox/rox/pkg/search"
)

// CountSortedSearcher generates a searcher that applies derived field sorting.
func CountSortedSearcher(searcher search.Searcher, counters map[string]counter.DerivedFieldCounter) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// If pagination not set, just skip.
			if q.GetPagination() == nil || len(q.GetPagination().GetSortOptions()) != 1 {
				return searcher.Search(ctx, q)
			}

			// If we cannot handle the derived field  sorting, skip it
			derivedFieldCounter, ok := counters[q.GetPagination().GetSortOptions()[0].GetField()]
			if !ok {
				return searcher.Search(ctx, q)
			}

			// Local copy to avoid changing input.
			local := q.Clone()

			// If derived field sort option is present, clear it
			sortOption := q.GetPagination().GetSortOptions()[0]
			local.Pagination.SortOptions = nil

			// Run the query without derived field sorting
			results, err := searcher.Search(ctx, local)
			if err != nil {
				return nil, err
			}

			ids := search.ResultsToIDs(results)
			idToCounts, err := derivedFieldCounter.Count(ctx, ids...)
			if err != nil {
				return nil, err
			}

			// Sort on derived field
			if sortOption.GetReversed() {
				sort.SliceStable(results, func(i, j int) bool {
					return idToCounts[results[j].ID] < (idToCounts[results[i].ID])
				})
			} else {
				sort.SliceStable(results, func(i, j int) bool {
					return idToCounts[results[i].ID] < (idToCounts[results[j].ID])
				})
			}
			return results, nil
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return searcher.Count(ctx, q)
		},
	}
}
