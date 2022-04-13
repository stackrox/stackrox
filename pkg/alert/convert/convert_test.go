package convert

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestAlertAndListAlertResourceTypesAreInSync(t *testing.T) {
	assert.Equal(t, storage.ListAlert_ResourceType_name[0], "DEPLOYMENT")
	assert.Equal(t, storage.Alert_Resource_ResourceType_name[0], "UNKNOWN")

	assert.Equal(t, len(storage.Alert_Resource_ResourceType_value), len(storage.ListAlert_ResourceType_value))
	for i, at := range storage.Alert_Resource_ResourceType_name {
		if r := storage.Alert_Resource_ResourceType(i); r == storage.Alert_Resource_UNKNOWN {
			continue
		}
		assert.Contains(t, storage.ListAlert_ResourceType_value, at)
		assert.Equal(t, at, storage.ListAlert_ResourceType_name[i])
	}
}
