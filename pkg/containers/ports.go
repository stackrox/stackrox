package containers

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	exposureOrder = []storage.PortConfig_ExposureLevel{storage.PortConfig_UNSET, storage.PortConfig_INTERNAL, storage.PortConfig_HOST, storage.PortConfig_NODE, storage.PortConfig_EXTERNAL}
	exposureRank  = utils.Invert(exposureOrder).(map[storage.PortConfig_ExposureLevel]int)
)

// CompareExposureLevel compares two exposure levels.
func CompareExposureLevel(a, b storage.PortConfig_ExposureLevel) int {
	aRank, ok := exposureRank[a]
	if !ok {
		errorhelpers.PanicOnDevelopmentf("invalid exposure level %v", a)
		aRank = -1
	}
	bRank, ok := exposureRank[b]
	if !ok {
		errorhelpers.PanicOnDevelopmentf("invalid exposure level %v", b)
		bRank = -1
	}
	return aRank - bRank
}
