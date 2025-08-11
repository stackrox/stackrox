//go:build test

package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(key string) *Client {
	return NewClient("test", "Test", "0.0.0", WithStorageKey(key))
}

func Test_noopClient(t *testing.T) {
	c := noopClient()
	assert.False(t, c.IsEnabled())
	assert.False(t, c.IsActive())
	assert.Equal(t, DisabledKey, c.storageKey.Get())
}

func TestNewClient(t *testing.T) {

	t.Run("incomplete config", func(t *testing.T) {
		t.Run("nil option", func(t *testing.T) {
			c := NewClient("", "", "", nil)
			require.NotNil(t, c)
			assert.False(t, c.IsEnabled())
			require.Equal(t, DisabledKey, c.storageKey.Get())
		})
		t.Run("no options", func(t *testing.T) {
			c := NewClient("", "", "")
			require.NotNil(t, c)
			assert.False(t, c.IsEnabled())
			require.Equal(t, DisabledKey, c.storageKey.Get())
		})
	})

	t.Run("dev main version", func(t *testing.T) {
		defer testutils.SetMainVersion(t, version.GetMainVersion())
		// Dev main version disables the client with empty key.
		testutils.SetMainVersion(t, "4.8.x-dev")

		t.Run("no key", func(t *testing.T) {
			// No-op in debug.
			c := newTestClient("")
			assert.False(t, c.IsEnabled())
			assert.False(t, c.IsActive()) // Won't hang, because disabled.
			c.consented.Set(true)         // Won't activate, because disabled.
			assert.False(t, c.IsActive())
			assert.Equal(t, DisabledKey, c.GetStorageKey())
		})
		t.Run("disabled key", func(t *testing.T) {
			// No-op in debug.
			c := newTestClient(DisabledKey)
			assert.False(t, c.IsEnabled())
			assert.False(t, c.IsActive()) // Won't hang, because disabled.
			c.consented.Set(true)         // Won't activate, because disabled.
			assert.False(t, c.IsActive())
			assert.Equal(t, DisabledKey, c.GetStorageKey())
		})
		t.Run("empty key", func(t *testing.T) {
			// No-op in debug.
			c := newTestClient("")
			assert.False(t, c.IsEnabled())
			c.consented.Set(true) // Won't activate, because disabled.
			assert.False(t, c.IsActive())
		})
		t.Run("with key", func(t *testing.T) {
			// Enabled, inactive.
			c := newTestClient("test-key")
			assert.True(t, c.IsEnabled())
			assert.False(t, c.consented.IsSet())
			c.consented.Set(true) // Now activates.
			assert.True(t, c.IsActive())
		})
	})

	t.Run("release main version", func(t *testing.T) {
		defer testutils.SetMainVersion(t, version.GetMainVersion())
		// testutils require `-tags test`, which makes the binary non-release.
		// So the client will stay no-op.
		testutils.SetMainVersion(t, "4.8.0")

		c := newTestClient("")
		assert.False(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		c.consented.Set(true)
		assert.False(t, c.IsActive())
	})
}

func TestClient_IsEnabled(t *testing.T) {
	var c *Client
	assert.False(t, c.IsEnabled())

	c = &Client{}
	assert.True(t, c.IsEnabled(), "should be temporarily enabled")

	c = &Client{storageKey: eventual.New[string]()}
	assert.True(t, c.IsEnabled(), "should be temporarily enabled")

	c.storageKey.Set("test-key")
	assert.True(t, c.IsEnabled())

	c.storageKey.Set(DisabledKey)
	assert.False(t, c.IsEnabled())
}

func TestClient_IsActive(t *testing.T) {
	c := &Client{
		storageKey: eventual.Now("test-key"),
		telemeter:  &nilTelemeter{},
		gatherer:   &nilGatherer{},
		consented:  eventual.Now(false),
	}
	assert.True(t, c.IsEnabled())

	assert.False(t, c.IsActive())
	c.WithdrawConsent()
	assert.False(t, c.IsActive())

	c.GrantConsent()
	assert.True(t, c.IsActive())
	c.GrantConsent()
	assert.True(t, c.IsActive())

	c.WithdrawConsent()
	assert.False(t, c.IsActive())

	assert.True(t, c.IsEnabled())
}

func TestClient_String(t *testing.T) {
	assert.Equal(t,
		`endpoint: "", initial key: "", configURL: "",`+
			` client ID: "", client type: "", client version: "",`+
			` await initial identity: false,`+
			` groups: map[], gathering period: 0s,`+
			` batch size: 0, push interval: 0s,`+
			` effective key: DISABLED, consent: false, identity sent: false`,
		NewClient("", "", "").String())

	c := NewClient(
		"id", "type", "version",
		WithEndpoint("endpoint"),
		WithStorageKey("key"),
		WithConfigURL("url"),
		WithAwaitInitialIdentity(),
		WithGroup("g1", "gid1"),
		WithGroup("g2", "gid2"),
		WithGatheringPeriod(5*time.Minute),
	)
	c.GrantConsent()
	c.InitialIdentitySent()
	assert.Equal(t,
		`endpoint: "endpoint", initial key: "key", configURL: "url",`+
			` client ID: "id", client type: "type", client version: "version",`+
			` await initial identity: true,`+
			` groups: map[g1:[gid1] g2:[gid2]], gathering period: 5m0s,`+
			` batch size: 0, push interval: 0s,`+
			` effective key: key, consent: true, identity sent: true`,
		c.String())
}

func TestClient_Reconfigure(t *testing.T) {
	var runtimeMux sync.Mutex
	var lastRC *RuntimeConfig

	newInactiveClient := func(key, url string) *Client {
		c := newClientFromConfig(&config{
			storageKey: key,
			configURL:  url,
			onReconfigure: func(rc *RuntimeConfig) {
				runtimeMux.Lock()
				defer runtimeMux.Unlock()
				lastRC = rc
			}})
		c.telemeter = &nilTelemeter{}
		c.consented = eventual.Now(false)
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
		c := newInactiveClient(DisabledKey, server.URL)
		err := c.Reconfigure()
		assert.ErrorIs(t, err, errox.InvalidArgs)
		assert.False(t, c.IsEnabled())
		assert.False(t, c.IsActive())
	})

	t.Run("reconfigure test key", func(t *testing.T) {
		lastRC = nil
		// Enabled client as some key is provided.
		c := newInactiveClient("some key", server.URL)
		assert.True(t, c.IsEnabled())
		err := c.Reconfigure()
		assert.Nil(t, err)
		assert.True(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		assert.ElementsMatch(t, APICallCampaign{
			MethodPattern("{put,delete}"),
			HeaderPattern("Accept-Encoding", "*json*"),
		}, lastRC.APICallCampaign)
	})

	t.Run("reconfigure empty key", func(t *testing.T) {
		c := newInactiveClient("", server.URL)
		assert.True(t, c.IsEnabled())
		assert.NoError(t, c.Reconfigure())
		assert.True(t, c.IsEnabled())
		assert.False(t, c.IsActive(), "Reconfigure shouldn't enable the client")
	})

	t.Run("bad url", func(t *testing.T) {
		// No-op client in non-release due to empty key:
		c := newInactiveClient("some-key", ":bad url:")
		assert.True(t, c.IsEnabled())
		err := c.Reconfigure()
		assert.Contains(t, err.Error(), "missing protocol scheme")
		assert.True(t, c.IsEnabled(), "failed reconfigure shouldn't change the client")
		assert.False(t, c.IsActive(), "Reconfigure shouldn't enable the client")
	})

	t.Run("bad content", func(t *testing.T) {
		lastRC = nil
		setConfig("not Jason")
		c := newInactiveClient("some-key", server.URL)
		assert.True(t, c.IsEnabled())
		err := c.Reconfigure()
		assert.Contains(t, err.Error(), "invalid character")
		assert.True(t, c.IsEnabled(), "failed reconfigure shouldn't change the client")
		assert.False(t, c.IsActive(), "Reconfigure shouldn't activate the client")

		assert.Nil(t, lastRC)
	})

	t.Run("reconfigure with DISABLED key", func(t *testing.T) {
		setConfig(`{"storage_key_v1": "` + DisabledKey + `"}`)
		c := newInactiveClient("some key", server.URL)
		assert.True(t, c.IsEnabled())
		c.consented.Set(true)
		assert.True(t, c.IsActive())

		err := c.Reconfigure()
		assert.Nil(t, err)
		assert.False(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		assert.Nil(t, c.telemeter, "telemeter has to be reset for new key")
	})

}

func TestClient_Telemeter(t *testing.T) {
	t.Run("empty key", func(t *testing.T) {
		c := newClientFromConfig(&config{})
		assert.True(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		tm := c.Telemeter() // should return nilTelemeter.
		_, ok := tm.(*nilTelemeter)
		assert.True(t, ok)
	})

	t.Run("new telemeter on reconfigure", func(t *testing.T) {
		const remoteKey = "remote-key"

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
		}))
		defer server.Close()

		c := newClientFromConfig(&config{
			storageKey: "test-key",
			configURL:  server.URL,
		})
		assert.Nil(t, c.telemeter)
		c.GrantConsent() // make IsActive() return true.

		previousTelemeter := c.Telemeter()
		assert.NotNil(t, c.telemeter)

		// Reconfigure will reset the telemeter even if the key doesn't change
		// in non-prod.
		assert.NoError(t, c.Reconfigure())
		assert.Equal(t, "test-key", c.storageKey.Get())
		assert.Nil(t, c.telemeter)
		assert.NotEqual(t, previousTelemeter, c.Telemeter())
	})
}

func TestClient_isIdentitySent(t *testing.T) {
	c := newClientFromConfig(&config{
		awaitInitialIdentity: false,
	})
	assert.False(t, c.isIdentitySent())
	c.InitialIdentitySent()
	assert.False(t, c.isIdentitySent())

	c = newClientFromConfig(&config{
		awaitInitialIdentity: true,
	})
	assert.False(t, c.isIdentitySent())
	c.InitialIdentitySent()
	assert.True(t, c.isIdentitySent())
}
