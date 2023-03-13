package paginated

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var (
	// ViolationTimeSortOption is a search options for alerts
	ViolationTimeSortOption = &v1.QuerySortOption{
		Field:    search.ViolationTime.String(),
		Reversed: true,
	}
)
