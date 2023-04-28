package paginated

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// GetViolationTimeSortOption returns the commonly used violation time sort option
func GetViolationTimeSortOption() *v1.QuerySortOption {
	return &v1.QuerySortOption{
		Field:    search.ViolationTime.String(),
		Reversed: true,
	}
}
