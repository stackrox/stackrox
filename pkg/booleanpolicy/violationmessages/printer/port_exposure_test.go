package printer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestPortExposureUpToDate(t *testing.T) {
	for k, v := range storage.PortConfig_ExposureLevel_value {
		if storage.PortConfig_ExposureLevel(v) == storage.PortConfig_UNSET {
			continue
		}
		assert.Contains(t, portExposureToDescMap, k)
	}

}
