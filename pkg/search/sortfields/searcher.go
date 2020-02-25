package sortfields

import (
	"context"

	"github.com/gogo/protobuf/proto"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// TransformSortFields applies transformation to specially handled sort fields e.g. multi-word fields.
func TransformSortFields(searcher search.Searcher) search.Searcher {
	return search.Func(func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
		// If pagination not set, just skip.
		if q.GetPagination() == nil {
			return searcher.Search(ctx, q)
		}

		// Local copy to avoid changing input.
		local := proto.Clone(q).(*v1.Query)

		sortOptions := make([]*v1.QuerySortOption, 0, len(local.GetPagination().GetSortOptions()))
		// replace the multi-word fields with the correct multi-word sort field, if present.
		for _, sortOption := range local.GetPagination().GetSortOptions() {
			sortFieldMapperFunc, ok := SortFieldsMap[search.FieldLabel(sortOption.GetField())]
			if ok {
				sortOptions = append(sortOptions, sortFieldMapperFunc(sortOption)...)
			} else {
				sortOptions = append(sortOptions, sortOption)
			}
		}

		// update query pagination
		local.Pagination.SortOptions = sortOptions

		// run the search
		return searcher.Search(ctx, local)
	})
}
