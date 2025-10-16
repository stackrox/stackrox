package paginated

import (
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

// GetViolationTimeSortOption returns the commonly used violation time sort option
func GetViolationTimeSortOption() *v1.QuerySortOption {
	qso := &v1.QuerySortOption{}
	qso.SetField(search.ViolationTime.String())
	qso.SetReversed(true)
	return qso
}
