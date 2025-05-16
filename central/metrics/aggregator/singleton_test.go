package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_reloadVulnerabilityTrackerConfig(t *testing.T) {
	tracker, err := reloadVulnerabilityTrackerConfig(nil)
	assert.NotNil(t, tracker)
	assert.NoError(t, err)
}
