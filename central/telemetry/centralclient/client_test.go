package centralclient

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

// newClientWithFactory creates a new client using the provided factory function.
// This is a test-only utility that returns a new instance each time.
func newClientWithFactory(factory func(string) *centralClient) *centralClient {
	cfg := initializeClient(factory)
	if cfg == nil {
		// Return a disabled client for offline mode (needed for testing)
		return &centralClient{Client: &phonehome.Client{}}
	}
	return cfg
}

func Test_newCentralClient(t *testing.T) {
	// Test production client with no telemetry key - should still create a client but inactive
	c := newCentralClient("test-id")
	assert.False(t, c.IsActive()) // No storage key means inactive
	assert.False(t, c.IsEnabled())
	assert.NotNil(t, c.Config) // Production client should have config

	// Test production client with telemetry key
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "non-empty")
	c = newCentralClient("test-id")
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())
	assert.Equal(t, "test-id", c.GroupID)
}

func Test_newCentralClientForTests(t *testing.T) {
	// Test client should always be disabled regardless of environment
	c := newCentralClientForTests("test-id")
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())
	assert.Nil(t, c.Config) // Test client has no config

	// Test with empty instance ID
	c = newCentralClientForTests("")
	assert.False(t, c.IsActive())
	assert.False(t, c.IsEnabled())
	assert.Nil(t, c.Config)
}

func Test_newClientWithFactory(t *testing.T) {
	// newClientWithFactory should create new instances each time

	// Test using the test factory
	testClient1 := newClientWithFactory(newCentralClientForTests)
	assert.NotNil(t, testClient1)
	assert.False(t, testClient1.IsActive())

	// Calling again should return a NEW instance (not singleton behavior)
	testClient2 := newClientWithFactory(newCentralClientForTests)
	assert.NotNil(t, testClient2)
	assert.False(t, testClient2.IsActive())
	assert.NotSame(t, testClient1, testClient2) // Different instances

	// Test with production factory
	prodClient := newClientWithFactory(newCentralClient)
	assert.NotNil(t, prodClient)
	// Behavior depends on environment setup, but should be different from test clients
	assert.NotSame(t, testClient1, prodClient)
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
