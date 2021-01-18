package image

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChartTemplatesAvailable(t *testing.T) {
	_, err := GetCentralServicesChartTemplate()
	assert.NoError(t, err, "failed to load central services chart")
	_, err = GetSecuredClusterServicesChartTemplate()
	assert.NoError(t, err, "failed to load secured cluster services chart")
}
