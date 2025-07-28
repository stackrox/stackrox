package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	t.Run("dev main version", func(t *testing.T) {
		defer testutils.SetMainVersion(t, version.GetMainVersion())
		// Dev main version disables the client with empty key.
		testutils.SetMainVersion(t, "4.8.0-dev")

		// No key => inactvie.
		c := NewClient(&Config{StorageKey: ""})
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		c.enabled = true // Won't help.
		assert.False(t, c.IsEnabled())

		// Has key => active.
		c = NewClient(&Config{StorageKey: "test-key"})
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		c.enabled = true // Now helps.
		assert.True(t, c.IsEnabled())
	})

	t.Run("release main version", func(t *testing.T) {
		defer testutils.SetMainVersion(t, version.GetMainVersion())
		// Release main version enables the (release) client with empty key.
		// But the client will still be disabled in the unit tests.
		testutils.SetMainVersion(t, "4.8.0")

		c := NewClient(&Config{StorageKey: ""})
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		c.enabled = true // Won't help.
		assert.False(t, c.IsEnabled())
	})

	c := NewClient(&Config{StorageKey: ""})
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c = NewClient(&Config{StorageKey: DisabledKey})
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())

	c = NewClient(&Config{StorageKey: "test-key"})
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())
	c.enabled = true
	assert.True(t, c.IsEnabled())
}

func TestClient_IsEnabled(t *testing.T) {
	c := &Client{
		config: Config{
			StorageKey: "test-key",
		},
		telemeter: &nilTelemeter{},
		gatherer:  &nilGatherer{},
		enabled:   false,
	}
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

func TestClient_String(t *testing.T) {
	cfg := Config{}
	assert.Equal(t, "{ClientID: ClientName: ClientVersion: GroupType: GroupID: StorageKey:DISABLED Endpoint: PushInterval:0s BatchSize:0 GatherPeriod:0s}",
		NewClient(&cfg).String())
}

func TestClient_Reconfigure(t *testing.T) {
	c := &Client{
		telemeter: &nilTelemeter{},
		enabled:   false,
	}

	t.Run("reconfigure DisabledKey with empty key", func(t *testing.T) {
		c.config.StorageKey = DisabledKey
		rc, err := c.Reconfigure("", "")
		assert.Nil(t, rc)
		assert.Nil(t, err)
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
	})

	t.Run("reconfigure DisabledKey with test key", func(t *testing.T) {
		c.config.StorageKey = DisabledKey
		rc, err := c.Reconfigure("", "test key")
		assert.Nil(t, rc)
		assert.Nil(t, err)
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
	})

	t.Run("reconfigure empty disabled config with a test key", func(t *testing.T) {
		c.config.StorageKey = ""
		rc, err := c.Reconfigure("", "test-key")
		assert.Equal(t, &RuntimeConfig{Key: "test-key", APICallCampaign: nil}, rc)
		assert.Nil(t, err)
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled())
	})

	t.Run("reconfigure empty enabled config with a test key", func(t *testing.T) {
		c.config.StorageKey = ""
		c.enabled = true
		rc, err := c.Reconfigure("", "test-key")
		assert.Equal(t, &RuntimeConfig{Key: "test-key", APICallCampaign: nil}, rc)
		assert.Nil(t, err)
		assert.True(t, c.IsActive())
		assert.True(t, c.IsEnabled())
	})

	t.Run("reconfigure enabled config with empty key", func(t *testing.T) {
		c.config.StorageKey = "test-key"
		c.enabled = true
		rc, err := c.Reconfigure("", "")
		assert.NotNil(t, rc)
		assert.Nil(t, err)
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled())
	})

	t.Run("reconfigure DisabledKey with downloaded test key", func(t *testing.T) {
		c.config.StorageKey = DisabledKey
		c.enabled = false

		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())

		const remoteKey = "remote-key"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
		}))
		defer server.Close()

		rc, err := c.Reconfigure(server.URL, "test-key")
		assert.Nil(t, rc)
		assert.Nil(t, err)
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
	})
}
