package sorted

import (
	"context"
	"sort"

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
					indexQuery = q.Clone()
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
