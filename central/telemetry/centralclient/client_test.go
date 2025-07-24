package centralclient

import (
	"errors"
	"sync"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/defaults"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stretchr/testify/assert"
)

// resetSingleton resets the singleton state for testing different scenarios
func resetSingleton() {
	client = nil
	once = sync.Once{}
}

// testInstanceIDProvider is a test implementation that returns a fixed instance ID
type testInstanceIDProvider struct {
	instanceID string
	err        error
}

func (p *testInstanceIDProvider) GetInstanceID() (string, error) {
	return p.instanceID, p.err
}

// newTestProvider creates a test instance ID provider with the given ID
func newTestProvider(instanceID string) *testInstanceIDProvider {
	return &testInstanceIDProvider{instanceID: instanceID}
}

// newTestProviderWithError creates a test instance ID provider that returns an error
func newTestProviderWithError(err error) *testInstanceIDProvider {
	return &testInstanceIDProvider{err: err}
}

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

// newTestClientWithProvider creates a test client using the provider
func newTestClientWithProvider(instanceID string, provider instanceIDProvider) *centralClient {
	return newCentralClientWithProvider(instanceID, provider)
}

func Test_newCentralClient(t *testing.T) {
	// Test production client with explicit instance ID (no provider needed)
	c := newCentralClient("test-id")
	assert.True(t, c.IsActive())   // Has proper config, so active even with empty storage key
	assert.False(t, c.IsEnabled()) // But not enabled since no storage key
	assert.NotNil(t, c.Config)     // Production client should have config

	// Test production client with telemetry key using test provider
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "non-empty")
	c = newTestClientWithProvider("", newTestProvider("test-central-id"))
	assert.True(t, c.IsActive())
	assert.False(t, c.IsEnabled())
	assert.Equal(t, "test-central-id", c.GroupID)
	assert.Equal(t, "test-central-id", c.Config.ClientID)
}

func Test_newCentralClient_ProviderError(t *testing.T) {
	// Test what happens when the instance ID provider returns an error
	errorProvider := newTestProviderWithError(errors.New("database connection failed"))
	c := newTestClientWithProvider("", errorProvider)

	// Should return a disabled client when provider fails
	assert.NotNil(t, c)
	assert.False(t, c.IsActive()) // Should be inactive due to provider error
	assert.False(t, c.IsEnabled())
	assert.Nil(t, c.Config) // No config when provider fails
}

func Test_Singleton_EmptyStorageKey(t *testing.T) {
	// Reset singleton state for this test
	resetSingleton()

	// Test Singleton with no telemetry key - should create an inactive client
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "")

	s := Singleton()
	assert.NotNil(t, s)
	assert.False(t, s.IsActive()) // Singleton creates nil config when no storage key
	assert.False(t, s.IsEnabled())
	assert.Nil(t, s.Config) // No config when storage key is empty

	// Calling again should return the same instance
	s2 := Singleton()
	assert.Same(t, s, s2)
}

func Test_Singleton_ValidStorageKey(t *testing.T) {
	// Reset singleton state for this test
	resetSingleton()

	// Test Singleton with valid telemetry key using test provider
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "test-storage-key")

	// Temporarily replace the default provider with test provider
	originalProvider := defaultInstanceIDProvider
	defaultInstanceIDProvider = newTestProvider("test-central-id")
	defer func() { defaultInstanceIDProvider = originalProvider }()

	s := Singleton()
	assert.NotNil(t, s)
	assert.True(t, s.IsActive())                             // Should be active with valid storage key
	assert.False(t, s.IsEnabled())                           // Not enabled until explicitly enabled
	assert.NotNil(t, s.Config)                               // Should have proper config
	assert.Equal(t, "test-storage-key", s.Config.StorageKey) // Storage key should match
	assert.Equal(t, "test-central-id", s.Config.ClientID)    // Instance ID should match

	// Calling again should return the same instance
	s2 := Singleton()
	assert.Same(t, s, s2)
}

func Test_Singleton_DisabledStorageKey(t *testing.T) {
	// Reset singleton state for this test
	resetSingleton()

	// Test Singleton with "DISABLED" storage key using test provider
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "DISABLED")

	// Temporarily replace the default provider with test provider
	originalProvider := defaultInstanceIDProvider
	defaultInstanceIDProvider = newTestProvider("test-central-id")
	defer func() { defaultInstanceIDProvider = originalProvider }()

	s := Singleton()
	assert.NotNil(t, s)
	assert.False(t, s.IsActive())                         // Should be inactive when storage key is "DISABLED"
	assert.False(t, s.IsEnabled())                        // Should not be enabled
	assert.NotNil(t, s.Config)                            // Should have config, but marked as disabled
	assert.Equal(t, "DISABLED", s.Config.StorageKey)      // Storage key should be "DISABLED"
	assert.Equal(t, "test-central-id", s.Config.ClientID) // Instance ID should match

	// Calling again should return the same instance
	s2 := Singleton()
	assert.Same(t, s, s2)
}

func Test_Singleton_OfflineMode(t *testing.T) {
	// Reset singleton state for this test
	resetSingleton()

	// Test Singleton in offline mode - should return nil
	t.Setenv(env.TelemetryStorageKey.EnvVar(), "valid-key")
	t.Setenv(env.OfflineModeEnv.EnvVar(), "true")

	s := Singleton()
	assert.Nil(t, s) // Should return nil in offline mode

	// Calling again should still return nil
	s2 := Singleton()
	assert.Nil(t, s2)
}

func Test_newClientWithFactory(t *testing.T) {
	// newClientWithFactory should create new instances each time

	// Test using a simple factory that creates disabled clients
	disabledFactory := func(instanceId string) *centralClient {
		provider := newTestProvider("test-instance-id")
		return newTestClientWithProvider(instanceId, provider)
	}

	testClient1 := newClientWithFactory(disabledFactory)
	assert.NotNil(t, testClient1)
	// Active status depends on storage key and other config

	// Calling again should return a NEW instance (not singleton behavior)
	testClient2 := newClientWithFactory(disabledFactory)
	assert.NotNil(t, testClient2)
	assert.NotSame(t, testClient1, testClient2) // Different instances

	// Test with explicit instance ID to avoid provider call
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
