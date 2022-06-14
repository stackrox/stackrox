package containers

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	exposureOrder = []storage.PortConfig_ExposureLevel{
		storage.PortConfig_UNSET,
		storage.PortConfig_INTERNAL,
		storage.PortConfig_HOST,
		storage.PortConfig_NODE,
		storage.PortConfig_ROUTE,
		storage.PortConfig_EXTERNAL,
	}
	exposureRank = utils.Invert(exposureOrder).(map[storage.PortConfig_ExposureLevel]int)
)

// CompareExposureLevel compares two exposure levels.
func CompareExposureLevel(a, b storage.PortConfig_ExposureLevel) int {
	aRank, ok := exposureRank[a]
	if !ok {
		utils.Should(errors.Errorf("invalid exposure level %v", a))
		aRank = -1
	}
	bRank, ok := exposureRank[b]
	if !ok {
		utils.Should(errors.Errorf("invalid exposure level %v", b))
		bRank = -1
	}
	return aRank - bRank
}
