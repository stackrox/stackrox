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

func Test_Singleton(t *testing.T) {
	testCases := []struct {
		name               string
		storageKey         string
		offlineMode        bool
		useProvider        bool
		providerInstanceID string
		expectNil          bool
		expectActive       bool
		expectEnabled      bool
		expectConfig       bool
		expectedStorageKey string
		expectedClientID   string
	}{
		{
			name:          "empty storage key",
			storageKey:    "",
			expectActive:  false,
			expectEnabled: false,
			expectConfig:  false, // nil config
		},
		{
			name:               "valid storage key",
			storageKey:         "test-storage-key",
			useProvider:        true,
			providerInstanceID: "test-central-id",
			expectActive:       true,
			expectEnabled:      false,
			expectConfig:       true,
			expectedStorageKey: "test-storage-key",
			expectedClientID:   "test-central-id",
		},
		{
			name:               "disabled storage key",
			storageKey:         "DISABLED",
			useProvider:        true,
			providerInstanceID: "test-central-id",
			expectActive:       false,
			expectEnabled:      false,
			expectConfig:       true,
			expectedStorageKey: "DISABLED",
			expectedClientID:   "test-central-id",
		},
		{
			name:        "offline mode",
			storageKey:  "valid-key",
			offlineMode: true,
			expectNil:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset singleton state for each test
			resetSingleton()

			// Set up environment variables
			t.Setenv(env.TelemetryStorageKey.EnvVar(), tc.storageKey)
			if tc.offlineMode {
				t.Setenv(env.OfflineModeEnv.EnvVar(), "true")
			}

			// Set up provider if needed
			var originalProvider instanceIDProvider
			if tc.useProvider {
				originalProvider = defaultInstanceIDProvider
				defaultInstanceIDProvider = newTestProvider(tc.providerInstanceID)
				defer func() { defaultInstanceIDProvider = originalProvider }()
			}

			// Call Singleton
			s := Singleton()

			// Assert expectations
			if tc.expectNil {
				assert.Nil(t, s, "expected Singleton to return nil")
				// Test second call also returns nil
				s2 := Singleton()
				assert.Nil(t, s2, "expected second Singleton call to return nil")
				return
			}

			// Non-nil assertions
			assert.NotNil(t, s, "expected Singleton to return non-nil client")
			assert.Equal(t, tc.expectActive, s.IsActive(), "IsActive() mismatch")
			assert.Equal(t, tc.expectEnabled, s.IsEnabled(), "IsEnabled() mismatch")

			if tc.expectConfig {
				assert.NotNil(t, s.Config, "expected non-nil config")
				if tc.expectedStorageKey != "" {
					assert.Equal(t, tc.expectedStorageKey, s.Config.StorageKey, "storage key mismatch")
				}
				if tc.expectedClientID != "" {
					assert.Equal(t, tc.expectedClientID, s.Config.ClientID, "client ID mismatch")
				}
			} else {
				assert.Nil(t, s.Config, "expected nil config")
			}

			// Test singleton behavior - second call should return same instance
			s2 := Singleton()
			assert.Same(t, s, s2, "expected Singleton to return same instance on second call")
		})
	}
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
