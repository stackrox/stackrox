package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsActive(t *testing.T) {
	c := &Client{nil, false}
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c.Config = &Config{}
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c.Config = &Config{
		StorageKey: "test-key",
	}
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c.Config = &Config{
		StorageKey: DisabledKey,
	}
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())
}
