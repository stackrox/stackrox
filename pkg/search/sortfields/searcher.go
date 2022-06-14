package sortfields

import (
	"context"

	v1 "github.com/stackrox/stackrox/generated/api/v1"
	"github.com/stackrox/stackrox/pkg/search"
)

// TransformSortFields applies transformation to specially handled sort fields e.g. multi-word fields.
func TransformSortFields(searcher search.Searcher, optionsMap search.OptionsMap) search.Searcher {
	return search.FuncSearcher{
		SearchFunc: func(ctx context.Context, q *v1.Query) ([]search.Result, error) {
			// If pagination not set, just skip.
			if q.GetPagination() == nil {
				return searcher.Search(ctx, q)
			}

			// Local copy to avoid changing input.
			local := q.Clone()

			sortOptions := make([]*v1.QuerySortOption, 0, len(local.GetPagination().GetSortOptions()))
			// replace the multi-word fields with the correct multi-word sort field, if present.
			for _, sortOption := range local.GetPagination().GetSortOptions() {
				sortFieldMapperFunc, ok := SortFieldsMap[search.FieldLabel(sortOption.GetField())]
				if !ok {
					sortOptions = append(sortOptions, sortOption)
					continue
				}

				transformedFields := sortFieldMapperFunc(sortOption)

				var anyTransformedFieldNotFound bool
				for _, transformedField := range transformedFields {
					if _, exists := optionsMap.Get(transformedField.GetField()); !exists {
						anyTransformedFieldNotFound = true
						break
					}
				}

				if anyTransformedFieldNotFound {
					sortOptions = append(sortOptions, sortOption)
				} else {
					sortOptions = append(sortOptions, transformedFields...)
				}
			}

			// update query pagination
			local.Pagination.SortOptions = sortOptions

			// run the search
			return searcher.Search(ctx, local)
		},
		CountFunc: func(ctx context.Context, q *v1.Query) (int, error) {
			return searcher.Count(ctx, q)
		},
	}
}
