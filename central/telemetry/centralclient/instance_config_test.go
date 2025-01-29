package centralclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getKey(t *testing.T) {
	cfg, err := getRuntimeConfig()
	// Telemetry should be disabled in test environment.
	assert.Nil(t, cfg)
	assert.NoError(t, err)
}

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

func TestInstanceConfig(t *testing.T) {
	const devVersion = "4.4.1-dev"
	const remoteKey = "remotekey"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"storage_key_v1": "` + remoteKey + `" }`))
	}))
	defer server.Close()

	testutils.SetMainVersion(t, devVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")

	tests := map[string]struct {
		configURL string
		key       string

		telemetryEnabled bool
		expectedKey      string
	}{
		"no URL, with key": {
			"", "whatever-key",
			true, "whatever-key",
		},
		"hardcoded URL, with key": {
			// non-release builds should provide the same key as in the remote
			// config to make the remote key effective, otherwise the provided
			// key is used instead.
			"hardcoded", "not-equal-to-hardcoded",
			true, "not-equal-to-hardcoded",
		},
		"custom URL, no key": {
			server.URL, "",
			false, "",
		},
		"custom URL, with matching key": {
			server.URL, remoteKey,
			true, remoteKey,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.TelemetryConfigURL.EnvVar(), test.configURL)
			t.Setenv(env.TelemetryStorageKey.EnvVar(), test.key)
			runtimeCfg, err := getRuntimeConfig()
			assert.NoError(t, err)
			if test.telemetryEnabled {
				cfg, props := getInstanceConfig("test-id", runtimeCfg.Key)
				require.NotNil(t, cfg, "Telemetry must be enabled")
				assert.Equal(t, test.expectedKey, cfg.StorageKey)
				assert.Equal(t, "test-id", cfg.ClientID)
				assert.Equal(t, devVersion, props["Central version"])
			} else {
				assert.Nil(t, runtimeCfg, "Telemetry must be disabled")
			}
		})
	}
}
