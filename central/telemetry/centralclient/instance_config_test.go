package centralclient

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_getKey(t *testing.T) {
	key, err := getKey()
	// Telemetry should be disabled in test environment.
	assert.Empty(t, key)
	assert.NoError(t, err)
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
			key, err := getKey()
			assert.NoError(t, err)
			if test.telemetryEnabled {
				cfg, props := getInstanceConfig("test-id", key)
				require.NotNil(t, cfg, "Telemetry must be enabled")
				assert.Equal(t, test.expectedKey, cfg.StorageKey)
				assert.Equal(t, "test-id", cfg.ClientID)
				assert.Equal(t, devVersion, props["Central version"])
			} else {
				assert.Empty(t, key, "Telemetry must be disabled")
			}
		})
	}
}

func Test_readExtraTelemetryCampaignFile(t *testing.T) {
	dir := t.TempDir()
	filename := path.Join(dir, "extraTelemetryCampaign.json")
	t.Setenv(telemetryCampaignFile.EnvVar(), filename)
	require.NoError(t, os.WriteFile(telemetryCampaignFile.Setting(), []byte(`[
	{"userAgents": ["test"], "codes": [500]},
	{"methods": ["delete"], "pathPatterns": ["/v1/*"]}
	]`), 0666))
	assert.Len(t, telemetryCampaign, 4, "built-in campaign")
	require.NoError(t, readExtraTelemetryCampaignFile(filename))
	require.Len(t, telemetryCampaign, 6)
	assert.Equal(t, "test", telemetryCampaign[4].UserAgents[0])
	assert.Equal(t, "/v1/*", telemetryCampaign[5].PathPatterns[0])
}
