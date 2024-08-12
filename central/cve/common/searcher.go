package common

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// WithoutOrphanedCVEsByDefault adds the OrphanedCVEs = false filter by default unless the passed query explicitly
// provides its own OrphanedCVEs filter
func WithoutOrphanedCVEsByDefault(searcher search.Searcher) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			q = interceptQuery(q)
			return searcher.Search(ctx, q)
		},

		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			q = interceptQuery(q)
			return searcher.Count(ctx, q)
		},
	}
}

func interceptQuery(q *v1.Query) *v1.Query {
	var isExplicitSearchByOrphaned bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		mfQ, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if ok && mfQ.MatchFieldQuery.GetField() == search.CVEOrphaned.String() {
			isExplicitSearchByOrphaned = true
			return
		}
	})
	if isExplicitSearchByOrphaned {
		return q
	}

	pagination := q.GetPagination()
	q.Pagination = nil
	ret := search.ConjunctionQuery(q, search.NewQueryBuilder().AddBools(search.CVEOrphaned, false).ProtoQuery())
	ret.Pagination = pagination
	return ret
}
