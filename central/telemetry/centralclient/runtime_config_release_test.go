//go:build release && test

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

func Test_getKey_release(t *testing.T) {
	testutils.SetMainVersion(t, "4.5.0")
	// Self-managed key has to be set to bypass the test build check.
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "eDd6QP8uWm0jCkAowEvijOPgeqtlulwR")
	cfg, err := getRuntimeConfig()
	// Telemetry should be enabled in release environment.
	assert.NotNil(t, cfg)
	assert.NoError(t, err)
}

func Test_reloadConfig_release(t *testing.T) {
	const releaseVersion = "4.4.1"
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
	testutils.SetMainVersion(t, releaseVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")
	t.Setenv(env.TelemetryConfigURL.EnvVar(), server.URL)
	t.Setenv(env.TelemetryStorageKey.EnvVar(), phonehome.DisabledKey)

	t.Run("ignore remote if local DISABLED", func(t *testing.T) {
		rc, err := getRuntimeConfig()
		require.NoError(t, err)
		require.Nil(t, rc)
		enable, err := applyConfig()
		require.NoError(t, err)
		assert.False(t, enable)
		assert.False(t, config.Enabled())
	})
}
