package centralclient

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/version"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

func Test_newCentralClient(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		c := newCentralClient("test-id")
		// Configuration has an empty key, but not running a release version,
		// therefore the client is not active.
		assert.False(t, c.IsEnabled())
		// Telemetry should be disabled in test environment with no key provided.
		assert.False(t, c.IsActive())
	})

	t.Run("with a key in env", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "non-empty")
		c := newCentralClient("test-id")
		assert.True(t, c.IsEnabled())
		// c.IsEnabled() will wait until the client is enabled
		// or disabled explicitly.
		assert.Equal(t,
			`endpoint: "https://console.redhat.com/connections/api",`+
				` key: "non-empty", configURL: "hardcoded",`+
				` client ID: "test-id", client type: "Central", client version: "`+version.GetMainVersion()+`",`+
				` await initial identity: true,`+
				` groups: map[Tenant:[test-id]], gathering period: 0s,`+
				` batch size: 1, push interval: 10m0s,`+
				` consent: <not set>, identity sent: true`,
			c.String())
	})

	t.Run("offline", func(t *testing.T) {
		t.Setenv(env.OfflineModeEnv.EnvVar(), "true")
		c := newCentralClient("test-id")
		assert.False(t, c.IsEnabled())
		assert.False(t, c.IsActive())
		assert.Equal(t, `endpoint: "", key: "DISABLED", configURL: "",`+
			` client ID: "", client type: "", client version: "",`+
			` await initial identity: false,`+
			` groups: map[], gathering period: 0s,`+
			` batch size: 0, push interval: 0s,`+
			` consent: false, identity sent: false`,
			c.String())
	})
}

func Test_getCentralDeploymentProperties(t *testing.T) {
	const devVersion = "4.4.1-dev"
	testutils.SetMainVersion(t, devVersion)
	t.Setenv(defaults.ImageFlavorEnvName, "opensource")

	props := getCentralDeploymentProperties()
	assert.Equal(t, map[string]any{
		"Central version":    "4.4.1-dev",
		"Chart version":      "400.4.1-dev",
		"Image Flavor":       "opensource",
		"Kubernetes version": "unknown",
		"Managed":            false,
		"Orchestrator":       "KUBERNETES_CLUSTER",
	}, props)
}
