//go:build release && !test

package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_release(t *testing.T) {
	require.True(t, version.IsReleaseVersion(),
		`must be run with, e.g., `+
			`-tags release -ldflags "-X github.com/stackrox/rox/pkg/version/internal.MainVersion=4.8.0"`)

	const remoteKey = "remote-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
		t.Log("served", remoteKey)
	}))
	defer server.Close()

	t.Run("no key", func(t *testing.T) {
		// This is a self-managed installation case.
		c := NewClient("test", "Test", "0.0.0",
			WithStorageKey(""),
			WithEndpoint(server.URL),
			WithConfigURL(server.URL),
			WithConfigureCallback(func(rc *RuntimeConfig) {
				t.Logf("reconfigured with %v", rc)
			}),
		)
		assert.True(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		assert.Equal(t, remoteKey, c.GetStorageKey(), "should fetch the key")
	})

	t.Run("DISABLED key", func(t *testing.T) {
		// This is release CI and infra clusters case.
		c := NewClient("test", "Test", "0.0.0",
			WithStorageKey(DisabledKey),
			WithEndpoint(server.URL),
			WithConfigURL(server.URL),
			WithConfigureCallback(func(rc *RuntimeConfig) {
				t.Logf("reconfigured with %v", rc)
			}),
		)
		assert.False(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		assert.Equal(t, DisabledKey, c.GetStorageKey())
	})
}
