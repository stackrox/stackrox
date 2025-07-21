package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_IsEnabled(t *testing.T) {
	c := &Client{&Config{
		StorageKey: "test-key",
		telemeter:  &nilTelemeter{},
		gatherer:   &nilGatherer{},
	}, false}
	assert.True(t, c.IsActive())

	assert.False(t, c.IsEnabled())
	c.Disable()
	assert.False(t, c.IsEnabled())

	c.Enable()
	assert.True(t, c.IsEnabled())
	c.Enable()
	assert.True(t, c.IsEnabled())

	c.Disable()
	assert.False(t, c.IsEnabled())

	assert.True(t, c.IsActive())
}

func TestClient_Reconfigure(t *testing.T) {
	cfg := &Client{&Config{
		telemeter: &nilTelemeter{},
	}, false}

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
