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

func Test_centralConfig_Reload_release(t *testing.T) {
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

	testutils.SetMainVersion(t, releaseVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")
	t.Setenv(env.TelemetryConfigURL.EnvVar(), server.URL)
	t.Setenv(env.TelemetryStorageKey.EnvVar(), phonehome.DisabledKey)

	cfg := makeCentralConfig("test-id")

	t.Run("ignore remote if local DISABLED", func(t *testing.T) {
		require.NoError(t, cfg.Reload())
		assert.False(t, cfg.IsEnabled())
		assert.False(t, cfg.IsValid())
	})
}
