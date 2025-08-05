package centralclient

import (
	"fmt"
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
		assert.False(t, c.IsActive())
		// Telemetry should be disabled in test environment with no key provided.
		assert.False(t, c.IsEnabled())
	})

	t.Run("with a key in env", func(t *testing.T) {
		t.Setenv(env.TelemetryStorageKey.EnvVar(), "non-empty")
		c := newCentralClient("test-id")
		assert.True(t, c.IsActive())
		// c.IsEnabled() will wait until the client is enabled
		// or disabled explicitly.
		assert.Equal(t, "{ClientID:test-id ClientName:Central ClientVersion:"+version.GetMainVersion()+
			" GroupType:Tenant GroupID:test-id StorageKey:non-empty"+
			" Endpoint:https://console.redhat.com/connections/api PushInterval:10m0s BatchSize:1 GatherPeriod:0s"+
			" ConfigURL:hardcoded OnReconfigure:"+fmt.Sprintf("%p", c.onReconfigure)+"}",
			c.String())
	})

	t.Run("offline", func(t *testing.T) {
		t.Setenv(env.OfflineModeEnv.EnvVar(), "true")
		c := newCentralClient("test-id")
		assert.False(t, c.IsActive())
		assert.False(t, c.IsEnabled())
		assert.Equal(t, "{ClientID: ClientName: ClientVersion: GroupType: GroupID: StorageKey:DISABLED"+
			" Endpoint: PushInterval:0s BatchSize:0 GatherPeriod:0s"+
			" ConfigURL: OnReconfigure:<nil>}",
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
