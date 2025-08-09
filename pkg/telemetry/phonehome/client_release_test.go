//go:build release && !test

package phonehome

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/eventual"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_release(t *testing.T) {
	require.True(t, version.IsReleaseVersion(),
		`must be run with, e.g., `+
			`-ldflags "-X github.com/stackrox/rox/pkg/version/internal.MainVersion=4.8.0"`)

	const remoteKey = "remote-key"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
		t.Log("served", remoteKey)
	}))
	defer server.Close()

	newTestClient := func(t *testing.T, key *eventual.Value[string]) *Client {
		return NewClient(&Config{
			StorageKey: key,
			ConfigURL:  server.URL,
			OnReconfigure: func(rc *RuntimeConfig) {
				t.Logf("reconfigured with %v", rc)
			},
		})
	}

	t.Run("no key", func(t *testing.T) {
		// This is a self-managed installation case.
		c := newTestClient(t, eventual.New[string]())
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		assert.Equal(t, remoteKey, c.GetStorageKey(), "should fetch the key")
	})

	t.Run("DISABLED key", func(t *testing.T) {
		// This is release CI and infra clusters case.
		c := newTestClient(t, eventual.Now(DisabledKey))
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		assert.Equal(t, DisabledKey, c.GetStorageKey())
	})

	t.Run("empty key", func(t *testing.T) {
		// An empty key also triggers periodic reconfiguration.
		c := newTestClient(t, eventual.Now(""))
		assert.True(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			assert.Equal(collect, remoteKey, c.GetStorageKey(), "should fetch the key")
		}, time.Second, time.Millisecond)
	})
}
