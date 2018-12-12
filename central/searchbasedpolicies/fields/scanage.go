package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

var scanAgeQueryBuilder = builders.NewDaysQueryBuilder(
	search.ImageScanTime,
	"Time of last scan",
	func(fields *storage.PolicyFields) (int64, bool) {
		if fields.GetSetScanAgeDays() == nil {
			return 0, false
		}
		return fields.GetScanAgeDays(), true
	},
)
