package centralclient

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInstanceConfig(t *testing.T) {
	cfg, props, err := getInstanceConfig()
	// Telemetry should be disabled in test environment.
	assert.Nil(t, cfg)
	assert.Nil(t, props)
	assert.Nil(t, err)
}
