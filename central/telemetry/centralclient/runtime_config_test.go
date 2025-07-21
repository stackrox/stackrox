package centralclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_centralConfig_Reload(t *testing.T) {
	const devVersion = "4.4.1-dev"
	const remoteKey = "remotekey"

	var runtimeConfigJSON string
	runtimeMux := sync.RWMutex{}
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
		runtimeMux.RLock()
		defer runtimeMux.RUnlock()
		_, _ = w.Write([]byte(runtimeConfigJSON))
	}))
	defer server.Close()

	testutils.SetMainVersion(t, devVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")
	t.Setenv(env.TelemetryConfigURL.EnvVar(), server.URL)
	t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)

	cfg := newCentralClient("test-id")

	t.Run("reload config with no changes", func(t *testing.T) {
		require.NoError(t, cfg.Reload())
		require.True(t, cfg.IsActive())
		assert.Equal(t, remoteKey, cfg.StorageKey)
		assert.Equal(t, append(permanentTelemetryCampaign,
			phonehome.MethodPattern("{put,delete}"),
			phonehome.HeaderPattern("Accept-Encoding", "*json*"),
		), cfg.telemetryCampaign)
	})

	t.Run("reload config with campaign changes", func(t *testing.T) {

		t.Setenv(env.TelemetryStorageKey.EnvVar(), "anotherKey")
		setConfig(`{"storage_key_v1": "anotherKey",
		"api_call_campaign": [
			{"method": "GET"},
			{"path": "*splunk*"}
		]}`)
		err := cfg.Reload()
		require.NoError(t, err)
		assert.True(t, cfg.IsActive())
		assert.Equal(t, "anotherKey", cfg.StorageKey)
		assert.Equal(t, append(permanentTelemetryCampaign,
			phonehome.MethodPattern("GET"),
			phonehome.PathPattern("*splunk*"),
		), cfg.telemetryCampaign)
	})
	t.Run("reload corrupted config", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "anotherKey")
		setConfig(`not JSON`)
		err := cfg.Reload()
		require.Equal(t, "cannot decode telemetry configuration: invalid character 'o' in literal null (expecting 'u')",
			err.Error())
		// The good config should be preserved.
		assert.True(t, cfg.IsActive())
		assert.Equal(t, "anotherKey", cfg.StorageKey)
		assert.Equal(t, append(permanentTelemetryCampaign,
			phonehome.MethodPattern("GET"),
			phonehome.PathPattern("*splunk*"),
		), cfg.telemetryCampaign)
	})
	t.Run("reload config with DISABLED key", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "DISABLED")
		setConfig(`{"storage_key_v1": "DISABLED",
		"api_call_campaign": [
			{"method": "GET"},
			{"path": "*splunk*"}
		]}`)
		require.NoError(t, cfg.Reload())
		assert.False(t, cfg.IsActive())
		assert.False(t, cfg.IsEnabled())
	})
	t.Run("reload when DISABLED", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)
		setConfig(`{"storage_key_v1": "` + remoteKey + `"}`)
		require.NoError(t, cfg.Reload())
		assert.False(t, cfg.IsEnabled())
		assert.False(t, cfg.IsActive(), "config should still be disabled")
	})

	t.Run("periodic reload", func(t *testing.T) {
		cfg.StorageKey = ""
		require.True(t, cfg.IsActive())
		require.False(t, cfg.IsEnabled())

		tickChan := make(chan time.Time)
		defer close(tickChan)

		go func() {
			for range tickChan {
				_ = cfg.Reload()
			}
		}()

		t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)
		setConfig(`{"storage_key_v1": "` + remoteKey + `"}
			"api_call_campaign": [{"method": "Test"}]}`)
		tickChan <- time.Now()
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			assert.Equal(collect, remoteKey, cfg.Config.GetStorageKey())
		}, 1*time.Second, 10*time.Millisecond)

		t.Setenv(env.TelemetryStorageKey.EnvVar(), "DISABLED")
		setConfig(`{"storage_key_v1": "DISABLED"}`)
		tickChan <- time.Now()
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			assert.False(collect, cfg.IsActive())
			assert.Equal(collect, phonehome.DisabledKey, cfg.Config.GetStorageKey())
		}, 1*time.Second, 10*time.Millisecond)
	})
}
