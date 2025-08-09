//go:build test

package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestConfig(key string) *Config {
	return &Config{StorageKey: eventual.Now(key)}
}

func TestNewClient(t *testing.T) {

	t.Run("incomplete config", func(t *testing.T) {
		t.Run("nil config", func(t *testing.T) {
			c := NewClient(nil)
			require.NotNil(t, c)
			assert.False(t, c.IsActive())
			require.Equal(t, DisabledKey, c.config.StorageKey.Get())
		})

		t.Run("nil key", func(t *testing.T) {
			// In non-release no key will deactivate the client.
			c := NewClient(&Config{})
			require.NotNil(t, c)
			require.NotNil(t, c.config.StorageKey)
			require.Equal(t, DisabledKey, c.config.StorageKey.Get())
			require.False(t, c.IsActive())
		})
	})

	t.Run("dev main version", func(t *testing.T) {
		defer testutils.SetMainVersion(t, version.GetMainVersion())
		// Dev main version disables the client with empty key.
		testutils.SetMainVersion(t, "4.8.x-dev")

		t.Run("no key", func(t *testing.T) {
			// No-op in debug.
			c := NewClient(&Config{StorageKey: eventual.New[string]()})
			assert.False(t, c.IsActive())
			assert.False(t, c.IsEnabled()) // Won't hang, because inactive.
			c.enabled.Set(true)            // Won't enable, because inactive.
			assert.False(t, c.IsEnabled())
		})

		t.Run("empty key", func(t *testing.T) {
			// No-op in debug.
			c := NewClient(newTestConfig(""))
			assert.False(t, c.IsActive())
			c.enabled.Set(true) // Won't enable, because inactive.
			assert.False(t, c.IsEnabled())
		})
		t.Run("with key", func(t *testing.T) {
			// Active, disabled.
			c := NewClient(newTestConfig("test-key"))
			assert.True(t, c.IsActive())
			assert.False(t, c.enabled.IsSet())
			c.enabled.Set(true) // Now enables.
			assert.True(t, c.IsEnabled())
		})
	})

	t.Run("release main version", func(t *testing.T) {
		defer testutils.SetMainVersion(t, version.GetMainVersion())
		// testutils require `-tags test`, which makes the binary non-release.
		// So the client will stay no-op.
		testutils.SetMainVersion(t, "4.8.0")

		c := NewClient(newTestConfig(""))
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		c.enabled.Set(true)
		assert.False(t, c.IsEnabled())
	})
}

func TestClient_IsActive(t *testing.T) {
	var c *Client
	assert.False(t, c.IsActive())

	c = &Client{}
	assert.True(t, c.IsActive(), "should be temporarily active")

	c.config.StorageKey = eventual.New[string]()
	assert.True(t, c.IsActive(), "should be temporarily active")

	c.config.StorageKey.Set("test-key")
	assert.True(t, c.IsActive())

	c.config.StorageKey.Set(DisabledKey)
	assert.False(t, c.IsActive())
}

func TestClient_IsEnabled(t *testing.T) {
	c := &Client{
		config:    *newTestConfig("test-key"),
		telemeter: &nilTelemeter{},
		gatherer:  &nilGatherer{},
		enabled:   eventual.Now(false),
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
	assert.Equal(t, "{ClientID: ClientName: ClientVersion: GroupType: GroupID: StorageKey:DISABLED "+
		"Endpoint: PushInterval:0s BatchSize:0 GatherPeriod:0s ConfigURL: OnReconfigure:<nil> Identified:<not set>}",
		NewClient(&cfg).String())
}

func TestClient_Reconfigure(t *testing.T) {
	var runtimeMux sync.Mutex
	var lastRC *RuntimeConfig

	newDisabledClient := func(key, url string) *Client {
		c := newOperationalClient(&Config{
			StorageKey: eventual.Now(key),
			ConfigURL:  url,
			OnReconfigure: func(rc *RuntimeConfig) {
				runtimeMux.Lock()
				defer runtimeMux.Unlock()
				lastRC = rc
			},
		})
		c.telemeter = &nilTelemeter{}
		c.enabled = eventual.Now(false)
		return c
	}

	const remoteKey = "remote-key"

	var runtimeConfigJSON string
	setConfig := func(cfg string) {
		runtimeMux.Lock()
		defer runtimeMux.Unlock()
		runtimeConfigJSON = cfg
	}
	setConfig(`{
		"storage_key_v1": "` + remoteKey + `",
		"api_call_campaign": [
			{"method": "{put,delete}"},
			{"headers": {"Accept-Encoding": "*json*"}}
		]
	}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		runtimeMux.Lock()
		defer runtimeMux.Unlock()
		_, _ = w.Write([]byte(runtimeConfigJSON))
	}))
	defer server.Close()

	t.Run("no reconfigure from DISABLED", func(t *testing.T) {
		c := newDisabledClient(DisabledKey, server.URL)
		err := c.Reconfigure()
		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
	})

	t.Run("reconfigure test key", func(t *testing.T) {
		lastRC = nil
		// Active client as some key is provided.
		c := newDisabledClient("some key", server.URL)
		assert.True(t, c.IsActive())
		err := c.Reconfigure()
		assert.Nil(t, err)
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		assert.ElementsMatch(t, APICallCampaign{
			MethodPattern("{put,delete}"),
			HeaderPattern("Accept-Encoding", "*json*"),
		}, lastRC.APICallCampaign)
	})

	t.Run("reconfigure empty key", func(t *testing.T) {
		c := newDisabledClient("", server.URL)
		assert.True(t, c.IsActive())
		assert.NoError(t, c.Reconfigure())
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled(), "Reconfigure shouldn't enable the client")
	})

	t.Run("bad url", func(t *testing.T) {
		// No-op client in non-release due to empty key:
		c := newDisabledClient("some-key", ":bad url:")
		assert.True(t, c.IsActive())
		err := c.Reconfigure()
		assert.Contains(t, err.Error(), "missing protocol scheme")
		assert.True(t, c.IsActive(), "failed reconfigure shouldn't change the client")
		assert.False(t, c.IsEnabled(), "Reconfigure shouldn't enable the client")
	})

	t.Run("bad content", func(t *testing.T) {
		lastRC = nil
		setConfig("not Jason")
		c := newDisabledClient("some-key", server.URL)
		assert.True(t, c.IsActive())
		err := c.Reconfigure()
		assert.Contains(t, err.Error(), "invalid character")
		assert.True(t, c.IsActive(), "failed reconfigure shouldn't change the client")
		assert.False(t, c.IsEnabled(), "Reconfigure shouldn't enable the client")

		assert.Nil(t, lastRC)
	})

	t.Run("reconfigure with DISABLED key", func(t *testing.T) {
		setConfig(`{"storage_key_v1": "` + DisabledKey + `"}`)
		c := newDisabledClient("some key", server.URL)
		assert.True(t, c.IsActive())
		c.enabled.Set(true)
		assert.True(t, c.IsEnabled())

		err := c.Reconfigure()
		assert.Nil(t, err)
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		assert.Nil(t, c.telemeter, "telemeter has to be reset for new key")
	})

}

func TestClient_Telemeter(t *testing.T) {
	const remoteKey = "remote-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
	}))
	defer server.Close()

	c := newOperationalClient(&Config{
		StorageKey: eventual.Now("test-key"),
		ConfigURL:  server.URL,
	})
	assert.Nil(t, c.telemeter)
	c.enabled.Set(true) // make IsEnabled() return true.

	tm1 := c.Telemeter()

	// Reconfigure won't reset telemeter as the key doesn't change in non-prod.
	assert.NoError(t, c.Reconfigure())
	assert.Equal(t, "test-key", c.config.StorageKey.Get())
	assert.NotNil(t, c.telemeter)
	tm2 := c.Telemeter()

	assert.Equal(t, tm1, tm2, "should be equal in non-prod")
}
