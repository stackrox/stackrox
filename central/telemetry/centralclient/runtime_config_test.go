package centralclient

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getRuntimeConfig(t *testing.T) {
	const devVersion = "4.4.1-dev"
	const remoteKey = "remotekey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"storage_key_v1": "` + remoteKey + `",
			"api_call_campaign": [
				{"method": "{put,delete}"},
				{"headers": {"Accept-Encoding": "*json*"}}
			]
		}`))
	}))
	defer server.Close()

	testutils.SetMainVersion(t, devVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")
	t.Setenv(env.TelemetryConfigURL.EnvVar(), server.URL)
	t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)
	cfg, err := getRuntimeConfig()
	require.NoError(t, err)
	assert.Equal(t, &phonehome.RuntimeConfig{
		Key: "remotekey",
		APICallCampaign: phonehome.APICallCampaign{
			{Method: phonehome.Pattern("{put,delete}").Ptr()},
			{Headers: map[string]phonehome.Pattern{"Accept-Encoding": phonehome.Pattern("*json*")}},
		},
	}, cfg)
}

func Test_reloadConfig(t *testing.T) {
	const devVersion = "4.4.1-dev"
	const remoteKey = "remotekey"

	var runtimeConfigJSON = `{
		"storage_key_v1": "` + remoteKey + `",
		"api_call_campaign": [
			{"method": "{put,delete}"},
			{"headers": {"Accept-Encoding": "*json*"}}
		]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(runtimeConfigJSON))
	}))
	defer server.Close()

	instanceId = "test-id"
	testutils.SetMainVersion(t, devVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")
	t.Setenv(env.TelemetryConfigURL.EnvVar(), server.URL)
	t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)

	t.Run("reload config with no changes", func(t *testing.T) {
		rc, err := getRuntimeConfig()
		require.NoError(t, err)
		enable, err := applyConfig()
		require.NoError(t, err)
		assert.True(t, enable)
		assert.Equal(t, rc.Key, config.StorageKey)
		assert.Equal(t, append(permanentTelemetryCampaign,
			&phonehome.APICallCampaignCriterion{Method: phonehome.Pattern("{put,delete}").Ptr()},
			&phonehome.APICallCampaignCriterion{Headers: map[string]phonehome.Pattern{"Accept-Encoding": phonehome.Pattern("*json*")}},
		), telemetryCampaign)
	})

	t.Run("reload config with campaign changes", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "anotherKey")
		runtimeConfigJSON = `{"storage_key_v1": "anotherKey",
		"api_call_campaign": [
			{"method": "GET"},
			{"path": "*splunk*"}
		]}`
		enable, err := applyConfig()
		require.NoError(t, err)
		assert.True(t, enable)
		assert.Equal(t, "anotherKey", config.StorageKey)
		assert.Equal(t, append(permanentTelemetryCampaign,
			&phonehome.APICallCampaignCriterion{Method: phonehome.Pattern("GET").Ptr()},
			&phonehome.APICallCampaignCriterion{Path: phonehome.Pattern("*splunk*").Ptr()},
		), telemetryCampaign)
	})
	t.Run("reload corrupted config", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "anotherKey")
		runtimeConfigJSON = `not JSON`
		enable, err := applyConfig()
		require.Error(t, err)
		assert.False(t, enable)
		assert.Equal(t, "anotherKey", config.StorageKey)
		assert.Equal(t, append(permanentTelemetryCampaign,
			&phonehome.APICallCampaignCriterion{Method: phonehome.Pattern("GET").Ptr()},
			&phonehome.APICallCampaignCriterion{Path: phonehome.Pattern("*splunk*").Ptr()},
		), telemetryCampaign)
	})
	t.Run("reload config with DISABLED key", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "DISABLED")
		runtimeConfigJSON = `{"storage_key_v1": "DISABLED",
		"api_call_campaign": [
			{"method": "GET"},
			{"path": "*splunk*"}
		]}`
		enable, err := applyConfig()
		require.NoError(t, err)
		assert.False(t, enable)
	})
	t.Run("reload when not enabled", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)
		runtimeConfigJSON = `{"storage_key_v1": "` + remoteKey + `"}`
		Enable()
		require.NoError(t, Reload())
		assert.True(t, enabled)
		assert.True(t, config.Enabled())
		Disable()
		assert.False(t, enabled)
		require.NoError(t, Reload())
		assert.False(t, enabled)
		assert.True(t, config.Enabled(), "config should still be good")
	})

	t.Run("periodic reload", func(t *testing.T) {
		tickChan := make(chan time.Time)
		defer close(tickChan)
		t.Setenv(env.TelemetryStorageKey.EnvVar(), remoteKey)
		runtimeConfigJSON = `{"storage_key_v1": "` + remoteKey + `"}
		"api_call_campaign": [{"method": "Test"}]}`
		Enable()
		defer Disable()
		assert.True(t, enabled)
		require.True(t, config.Enabled())

		go func() {
			for range tickChan {
				_ = Reload()
			}
		}()
		tickChan <- time.Now()
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			startMux.Lock()
			defer startMux.Unlock()
			assert.True(collect, enabled)
			assert.True(collect, config.Enabled())
		}, 1*time.Second, 10*time.Millisecond)
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "DISABLED")
		runtimeConfigJSON = `{"storage_key_v1": "DISABLED"}`
		tickChan <- time.Now()
		assert.EventuallyWithT(t, func(collect *assert.CollectT) {
			startMux.Lock()
			defer startMux.Unlock()
			assert.False(collect, enabled)
			assert.True(collect, config.Enabled())
		}, 1*time.Second, 10*time.Millisecond)
	})
}
