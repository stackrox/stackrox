package index

import (
	"context"

	"github.com/stackrox/rox/central/alert/index/internal/index"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/paginated"
)

var (
	defaultSortOption = &v1.SortOption{
		Field:    search.ViolationTime.String(),
		Reversed: true,
	}
)

type indexerImpl struct {
	index.Indexer
}

// Search takes a SearchRequest and finds any matches
func (b *indexerImpl) Search(q *v1.Query) ([]search.Result, error) {
	var querySpecifiesStateField bool
	search.ApplyFnToAllBaseQueries(q, func(bq *v1.BaseQuery) {
		matchFieldQuery, ok := bq.GetQuery().(*v1.BaseQuery_MatchFieldQuery)
		if !ok {
			return
		}
		if matchFieldQuery.MatchFieldQuery.GetField() == search.ViolationState.String() {
			querySpecifiesStateField = true
		}
	})

	// By default, set stale to false.
	if !querySpecifiesStateField {
		cq := search.ConjunctionQuery(q, search.NewQueryBuilder().AddStrings(search.ViolationState, storage.ViolationState_ACTIVE.String()).ProtoQuery())
		cq.Pagination = q.GetPagination()
		q = cq
	}

	return paginated.Paginated(
		paginated.WithDefaultSortOption(
			search.WrapContextLessSearcher(b.Indexer),
			defaultSortOption),
	).Search(context.TODO(), q)
}
