package sachelper

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/set"
)

func getSanitizedPagination(requested *v1.Pagination, allowedSortFields set.StringSet) *v1.QueryPagination {
	if requested == nil {
		return nil
	}
	sanitized := &v1.QueryPagination{}
	sanitized.SetLimit(requested.GetLimit())
	sanitized.SetOffset(requested.GetOffset())
	sanitized.SetSortOptions(nil)
	if requested.GetSortOption() != nil {
		sortField := requested.GetSortOption().GetField()
		if allowedSortFields.Contains(sortField) {
			sanitizedSortOption := &v1.QuerySortOption{}
			sanitizedSortOption.SetField(sortField)
			sanitizedSortOption.SetReversed(requested.GetSortOption().GetReversed())
			sanitizedSortOption.ClearSearchAfterOpt()
			sanitized.SetSortOptions(append(sanitized.GetSortOptions(), sanitizedSortOption))
		}
	}
	return sanitized
}
