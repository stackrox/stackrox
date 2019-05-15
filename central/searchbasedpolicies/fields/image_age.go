package fields

import (
	"github.com/stackrox/rox/central/searchbasedpolicies/builders"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
)

// ImageAgeQueryBuilder is a time based query builder on the age if an image used by a deployment.
var ImageAgeQueryBuilder = builders.NewDaysQueryBuilder(
	search.ImageCreatedTime,
	"Time of image creation",
	func(fields *storage.PolicyFields) (int64, bool) {
		if fields.GetSetImageAgeDays() == nil {
			return 0, false
		}
		return fields.GetImageAgeDays(), true
	},
)
