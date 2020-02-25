package sorted

import (
	"context"
	"sort"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// Ranker returns the rank for the given id for the given field.
//go:generate mockgen-wrapper
type Ranker interface {
	// GetRankForID returns the rank of the object referenced by the given ID.
	GetRankForID(from string) int64
}

// Searcher returns a Searcher that applies the sort for the custom field if it exists in the input query.
func Searcher(searcher search.Searcher, field search.FieldLabel, ranker Ranker) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		var indexQuery *v1.Query
		var sortByRank bool
		var reversed bool
		if q.GetPagination() != nil && len(q.GetPagination().GetSortOptions()) == 1 {
			if q.GetPagination().GetSortOptions()[0].GetField() == field.String() {
				indexQuery = proto.Clone(q).(*v1.Query)
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

		sort.SliceStable(results, func(i, j int) bool {
			if reversed {
				i, j = j, i
			}
			rankI := ranker.GetRankForID(results[i].ID)
			rankJ := ranker.GetRankForID(results[j].ID)
			return rankI < rankJ
		})
		return results, nil
	})
}
