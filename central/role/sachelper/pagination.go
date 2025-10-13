package sachelper

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

func getSanitizedPagination(requested *v1.Pagination, allowedSortFields set.StringSet) *v1.QueryPagination {
	if requested == nil {
		return nil
	}
	sanitized := &v1.QueryPagination{
		Limit:       requested.GetLimit(),
		Offset:      requested.GetOffset(),
		SortOptions: nil,
	}
	if requested.GetSortOption() != nil {
		sortField := requested.GetSortOption().GetField()
		if allowedSortFields.Contains(sortField) {
			sanitizedSortOption := &v1.QuerySortOption{
				Field:          sortField,
				Reversed:       requested.GetSortOption().GetReversed(),
				SearchAfterOpt: nil,
			}
			sanitized.SortOptions = append(sanitized.SortOptions, sanitizedSortOption)
		}
	}
	return sanitized
}
