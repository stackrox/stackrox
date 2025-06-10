package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_IsValid(t *testing.T) {
	var cfg *Config
	assert.False(t, cfg.IsActive())
	assert.False(t, cfg.IsEnabled())

	cfg = &Config{}
	assert.True(t, cfg.IsActive())
	assert.False(t, cfg.IsEnabled())

	cfg = &Config{
		StorageKey: "test-key",
	}
	assert.True(t, cfg.IsActive())
	assert.False(t, cfg.IsEnabled())

	cfg = &Config{
		StorageKey: DisabledKey,
	}
	assert.False(t, cfg.IsActive())
	assert.False(t, cfg.IsEnabled())
}

func TestConfig_IsEnabled(t *testing.T) {
	cfg := &Config{
		StorageKey: "test-key",
		telemeter:  &nilTelemeter{},
		gatherer:   &nilGatherer{},
	}
	assert.True(t, cfg.IsActive())

	assert.False(t, cfg.IsEnabled())
	cfg.Disable()
	assert.False(t, cfg.IsEnabled())

	cfg.Enable()
	assert.True(t, cfg.IsEnabled())
	cfg.Enable()
	assert.True(t, cfg.IsEnabled())

	cfg.Disable()
	assert.False(t, cfg.IsEnabled())

	assert.True(t, cfg.IsActive())
}

func TestConfig_Reconfigure(t *testing.T) {
	cfg := &Config{
		telemeter: &nilTelemeter{},
	}

	t.Run("reconfigure DisabledKey with empty key", func(t *testing.T) {
		cfg.StorageKey = DisabledKey
		rc, err := cfg.Reconfigure("", "")
		assert.Nil(t, rc)
		assert.Nil(t, err)
		assert.False(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())
	})

	t.Run("reconfigure DisabledKey with test key", func(t *testing.T) {
		cfg.StorageKey = DisabledKey
		rc, err := cfg.Reconfigure("", "test key")
		assert.Nil(t, rc)
		assert.Nil(t, err)
		assert.False(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())
	})

	t.Run("reconfigure empty disabled config with a test key", func(t *testing.T) {
		cfg.StorageKey = ""
		rc, err := cfg.Reconfigure("", "test-key")
		assert.Equal(t, &RuntimeConfig{Key: "test-key", APICallCampaign: nil}, rc)
		assert.Nil(t, err)
		assert.True(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())
	})

	t.Run("reconfigure empty enabled config with a test key", func(t *testing.T) {
		cfg.StorageKey = ""
		cfg.enabled = true
		rc, err := cfg.Reconfigure("", "test-key")
		assert.Equal(t, &RuntimeConfig{Key: "test-key", APICallCampaign: nil}, rc)
		assert.Nil(t, err)
		assert.True(t, cfg.IsActive())
		assert.True(t, cfg.IsEnabled())
	})

	t.Run("reconfigure enabled config with empty key", func(t *testing.T) {
		cfg.StorageKey = "test-key"
		cfg.enabled = true
		rc, err := cfg.Reconfigure("", "")
		assert.NotNil(t, rc)
		assert.Nil(t, err)
		assert.True(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())
	})

	t.Run("reconfigure DisabledKey with downloaded test key", func(t *testing.T) {
		cfg.StorageKey = DisabledKey
		cfg.enabled = false

		assert.False(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())

		const remoteKey = "remote-key"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
		}))
		defer server.Close()

		rc, err := cfg.Reconfigure(server.URL, "test-key")
		assert.Nil(t, rc)
		assert.Nil(t, err)
		assert.False(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())
	})
}
