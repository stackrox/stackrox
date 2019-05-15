package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ScanAgeQueryBuilder is a time based query builder on the age of an image's scan data.
var ScanAgeQueryBuilder = builders.NewDaysQueryBuilder(
	search.ImageScanTime,
	"Time of last scan",
	func(fields *storage.PolicyFields) (int64, bool) {
		if fields.GetSetScanAgeDays() == nil {
			return 0, false
		}
		return fields.GetScanAgeDays(), true
	},
)
