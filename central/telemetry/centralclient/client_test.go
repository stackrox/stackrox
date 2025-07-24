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
	// Test production client with no telemetry key - should still create an active client
	// An empty storage key doesn't make it inactive, only "DISABLED" does
	c := newCentralClient("test-id")
	assert.True(t, c.IsActive())   // Has proper config, so active even with empty storage key
	assert.False(t, c.IsEnabled()) // But not enabled since no storage key
	assert.NotNil(t, c.Config)     // Production client should have config

	// Test production client with telemetry key
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "non-empty")
	c = newCentralClient("test-id")
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())
	assert.Equal(t, "test-id", c.GroupID)
}

func Test_Singleton(t *testing.T) {
	// Test Singleton with no telemetry key - should create an inactive client
	// This demonstrates the difference between Singleton and newCentralClient
	s := Singleton()
	assert.False(t, s.IsActive()) // Singleton creates nil config when no storage key
	assert.False(t, s.IsEnabled())
	assert.Nil(t, s.Config) // No config when storage key is empty

	// Calling again should return the same instance
	s2 := Singleton()
	assert.Same(t, s, s2)
}

func Test_newClientWithFactory(t *testing.T) {
	// newClientWithFactory should create new instances each time

	// Test using a disabled test factory that avoids database access
	disabledFactory := func(instanceId string) *centralClient {
		if instanceId == "" {
			instanceId = "test-instance-id"
		}
		return &centralClient{Client: &phonehome.Client{}}
	}

	testClient1 := newClientWithFactory(disabledFactory)
	assert.NotNil(t, testClient1)
	assert.False(t, testClient1.IsActive())

	// Calling again should return a NEW instance (not singleton behavior)
	testClient2 := newClientWithFactory(disabledFactory)
	assert.NotNil(t, testClient2)
	assert.False(t, testClient2.IsActive())
	assert.NotSame(t, testClient1, testClient2) // Different instances

	// Test with production factory but provide instance ID to avoid DB access
	prodFactory := func(instanceId string) *centralClient {
		return newCentralClient("test-prod-id")
	}
	prodClient := newClientWithFactory(prodFactory)
	assert.NotNil(t, prodClient)
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
