package phonehome

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsValid(t *testing.T) {
	var cfg *Config
	assert.False(t, cfg.IsValid())
	assert.False(t, cfg.IsEnabled())

	cfg = &Config{}
	assert.True(t, cfg.IsValid())
	assert.False(t, cfg.IsEnabled())

	cfg = &Config{
		StorageKey: "test-key",
	}
	assert.True(t, cfg.IsValid())
	assert.False(t, cfg.IsEnabled())

	cfg = &Config{
		StorageKey: DisabledKey,
	}
	assert.False(t, cfg.IsValid())
	assert.False(t, cfg.IsEnabled())
}

func TestConfig_IsEnabled(t *testing.T) {
	cfg := &Config{
		StorageKey: "test-key",
		telemeter:  &nilTelemeter{},
		gatherer:   &nilGatherer{},
	}
	assert.True(t, cfg.IsValid())

	assert.False(t, cfg.IsEnabled())
	cfg.Disable()
	assert.False(t, cfg.IsEnabled())

	cfg.Enable()
	assert.True(t, cfg.IsEnabled())
	cfg.Enable()
	assert.True(t, cfg.IsEnabled())

	cfg.Disable()
	assert.False(t, cfg.IsEnabled())

	assert.True(t, cfg.IsValid())
}

func TestConfig_Reconfigure(t *testing.T) {
	cfg := &Config{
		StorageKey: DisabledKey,
		telemeter:  &nilTelemeter{},
	}

	rc, err := cfg.Reconfigure("", "")
	assert.Nil(t, rc)
	assert.Nil(t, err)
	assert.False(t, cfg.IsValid())

	cfg.StorageKey = ""
	rc, err = cfg.Reconfigure("", "test-key")
	assert.Equal(t, &RuntimeConfig{Key: "test-key", APICallCampaign: nil}, rc)
	assert.Nil(t, err)
	assert.True(t, cfg.IsValid())
}
