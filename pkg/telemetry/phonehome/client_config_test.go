package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Enabled(t *testing.T) {
	var cfg *Config
	assert.False(t, cfg.Enabled())

	cfg = &Config{}
	assert.False(t, cfg.Enabled())

	cfg = &Config{
		StorageKey: "test-key",
	}
	assert.True(t, cfg.Enabled())
}
