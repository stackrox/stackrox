package containers

import (
	"github.com/stackrox/rox/generated/storage"
)

// IncreasedExposureLevel returns whether the new level carries increased exposure.
func IncreasedExposureLevel(old, new storage.PortConfig_Exposure) bool {
	switch old {
	case storage.PortConfig_UNSET:
		return true
	case storage.PortConfig_INTERNAL:
		return new == storage.PortConfig_NODE || new == storage.PortConfig_EXTERNAL
	case storage.PortConfig_NODE:
		return new == storage.PortConfig_EXTERNAL
	default:
		return false
	}
}
