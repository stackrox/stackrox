package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsActive(t *testing.T) {
	var c *Client
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c = &Client{}
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c.config = Config{
		StorageKey: "test-key",
	}
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c.config = Config{
		StorageKey: DisabledKey,
	}
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())
}
