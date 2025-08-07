package sorted

import (
	"context"
	"sort"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Ranker returns the rank for the given id for the given field.
//
//go:generate mockgen-wrapper
type Ranker interface {
	// GetRankForID returns the rank of the object referenced by the given ID.
	GetRankForID(from string) int64
}

// Searcher returns a Searcher that applies the sort for the custom field if it exists in the input query.
func Searcher(searcher search.Searcher, field search.FieldLabel, ranker Ranker) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			var indexQuery *v1.Query
			var sortByRank bool
			var reversed bool
			if q.GetPagination() != nil && len(q.GetPagination().GetSortOptions()) == 1 {
				if q.GetPagination().GetSortOptions()[0].GetField() == field.String() {
					indexQuery = q.CloneVT()
					sortByRank = true
					reversed = indexQuery.GetPagination().GetSortOptions()[0].GetReversed()
					indexQuery.Pagination = nil
				}
			}
			if !sortByRank {
				indexQuery = q
			}

			results, err := searcher.Search(ctx, indexQuery)
			if err != nil || !sortByRank {
				return results, err
			}

			sort.Stable(&resultsSorter{
				results:  results,
				reversed: reversed,
				ranker:   ranker,
			})
			return results, nil
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return searcher.Count(ctx, q)
		},
	}
}

// IsValidPriorityQuery returns if the query has priority as a sort option.  If priority exists as a sort
// option but additional sort options are present, an error is returned.
func IsValidPriorityQuery(q *v1.Query, field search.FieldLabel) (bool, error) {
	if q.GetPagination() != nil && len(q.GetPagination().GetSortOptions()) == 1 {
		if q.GetPagination().GetSortOptions()[0].GetField() == field.String() {
			return true, nil
		}
	} else if q.GetPagination() != nil && len(q.GetPagination().GetSortOptions()) > 1 {
		for _, q := range q.GetPagination().GetSortOptions() {
			if q.GetField() == field.String() {
				return false, errors.Errorf("query field %v not supported with other sort options", field.String())
			}
		}
	}
	return false, nil
}

// RemovePrioritySortFromQuery removes the priority sort option from the query.  Priority is not an actual
// field.  So sorting by priority is done via the ranker.  Sort by priority is only supported if it is the only
// sort option.  The caller will have to make a subsequent call to `SortResults` after executing the query
// with priority removed.
func RemovePrioritySortFromQuery(q *v1.Query, field search.FieldLabel) (*v1.Query, bool, error) {
	validPriorityQuery, err := IsValidPriorityQuery(q, field)
	if err != nil {
		return nil, false, err
	}
	if !validPriorityQuery {
		return nil, false, errors.Errorf("query does not sort by %q", field.String())
	}

	var indexQuery *v1.Query
	var reversed bool
	if q.GetPagination() != nil && len(q.GetPagination().GetSortOptions()) == 1 {
		if q.GetPagination().GetSortOptions()[0].GetField() == field.String() {
			indexQuery = q.CloneVT()
			reversed = indexQuery.GetPagination().GetSortOptions()[0].GetReversed()
			indexQuery.Pagination = nil
		}
	}

	return indexQuery, reversed, nil
}

// SortResults sorts the result from a search based on the values held within the ranker.
func SortResults(results []search.Result, reversed bool, ranker Ranker) []search.Result {
	sort.Stable(&resultsSorter{
		results:  results,
		reversed: reversed,
		ranker:   ranker,
	})
	return results
}

type resultsSorter struct {
	results  []search.Result
	reversed bool
	ranker   Ranker
}

func (s *resultsSorter) Len() int {
	return len(s.results)
}

func (s *resultsSorter) Less(i, j int) bool {
	if s.reversed {
		i, j = j, i
	}
	rankI := s.ranker.GetRankForID(s.results[i].ID)
	rankJ := s.ranker.GetRankForID(s.results[j].ID)
	return rankI < rankJ
}

func (s *resultsSorter) Swap(i, j int) {
	s.results[i], s.results[j] = s.results[j], s.results[i]
}
