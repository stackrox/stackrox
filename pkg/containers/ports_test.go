package containers

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestCompareExposureLevel(t *testing.T) {
	t.Parallel()

	assert.True(t, CompareExposureLevel(storage.PortConfig_INTERNAL, storage.PortConfig_EXTERNAL) < 0)
	assert.True(t, CompareExposureLevel(storage.PortConfig_NODE, storage.PortConfig_INTERNAL) > 0)
	assert.True(t, CompareExposureLevel(storage.PortConfig_UNSET, storage.PortConfig_UNSET) == 0)
}
