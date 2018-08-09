package containers

import "github.com/stackrox/rox/generated/api/v1"

// IncreasedExposureLevel returns whether the new level carries increased exposure.
func IncreasedExposureLevel(old, new v1.PortConfig_Exposure) bool {
	switch old {
	case v1.PortConfig_UNSET:
		return true
	case v1.PortConfig_INTERNAL:
		return new == v1.PortConfig_NODE || new == v1.PortConfig_EXTERNAL
	case v1.PortConfig_NODE:
		return new == v1.PortConfig_EXTERNAL
	default:
		return false
	}
}
