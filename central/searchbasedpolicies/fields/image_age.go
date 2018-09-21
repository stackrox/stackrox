package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/search"
)

var imageAgeQueryBuilder = builders.NewDaysQueryBuilder(
	search.ImageCreatedTime,
	"Time of image creation",
	func(fields *v1.PolicyFields) (int64, bool) {
		if fields.GetSetImageAgeDays() == nil {
			return 0, false
		}
		return fields.GetImageAgeDays(), true
	},
)
