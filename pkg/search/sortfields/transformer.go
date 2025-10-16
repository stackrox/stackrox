package sortfields

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// TransformSortOptions applies transformation to specially handled sort fields e.g. multi-word fields.
func TransformSortOptions(q *v1.Query, optionsMap search.OptionsMap) *v1.Query {
	// If pagination not set, just skip.
	if q == nil || q.GetPagination() == nil || optionsMap == nil {
		return q
	}

	// Local copy to avoid changing input.
	local := q.CloneVT()

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
	local.GetPagination().SetSortOptions(sortOptions)

	return local
}
