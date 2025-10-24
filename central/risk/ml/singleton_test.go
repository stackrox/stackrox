package ml

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSingleton_Disabled(t *testing.T) {
	// Ensure ML service is disabled for this test
	os.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "false")
	defer os.Unsetenv("ROX_ML_RISK_SERVICE_ENABLED")

	// Reset singleton before test
	Reset()

	client := Singleton()

	// Should return noOpClient when disabled
	assert.NotNil(t, client)
	_, ok := client.(*noOpClient)
	assert.True(t, ok, "Should return noOpClient when ML service is disabled")

	// IsEnabled should return false
	assert.False(t, IsEnabled())

	// GetClientError should return nil since noOpClient doesn't error during init
	assert.NoError(t, GetClientError())
}

func TestSingleton_Enabled_InvalidEndpoint(t *testing.T) {
	// Enable ML service but with invalid endpoint
	os.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "true")
	os.Setenv("ROX_ML_RISK_SERVICE_ENDPOINT", "invalid-endpoint-that-does-not-exist:99999")
	os.Setenv("ROX_ML_RISK_SERVICE_TIMEOUT", "1s") // Short timeout for faster test
	defer func() {
		os.Unsetenv("ROX_ML_RISK_SERVICE_ENABLED")
		os.Unsetenv("ROX_ML_RISK_SERVICE_ENDPOINT")
		os.Unsetenv("ROX_ML_RISK_SERVICE_TIMEOUT")
	}()

	// Reset singleton before test
	Reset()

	client := Singleton()

	// Should return noOpClient when connection fails
	assert.NotNil(t, client)
	_, ok := client.(*noOpClient)
	assert.True(t, ok, "Should fallback to noOpClient when connection fails")

	// IsEnabled should return true (environment setting)
	assert.True(t, IsEnabled())

	// GetClientError should return the connection error
	err := GetClientError()
	assert.Error(t, err, "Should return connection error")
}

func TestSingleton_InvalidTimeout(t *testing.T) {
	// Enable ML service with invalid timeout
	os.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "true")
	os.Setenv("ROX_ML_RISK_SERVICE_ENDPOINT", "localhost:8080")
	os.Setenv("ROX_ML_RISK_SERVICE_TIMEOUT", "invalid-timeout")
	defer func() {
		os.Unsetenv("ROX_ML_RISK_SERVICE_ENABLED")
		os.Unsetenv("ROX_ML_RISK_SERVICE_ENDPOINT")
		os.Unsetenv("ROX_ML_RISK_SERVICE_TIMEOUT")
	}()

	// Reset singleton before test
	Reset()

	client := Singleton()

	// Should still create client (with default timeout)
	assert.NotNil(t, client)

	// IsEnabled should return true
	assert.True(t, IsEnabled())
}

func TestSingleton_TLSConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		tlsEnabled  string
		expectNoOp  bool
		description string
	}{
		{
			name:        "TLS disabled",
			tlsEnabled:  "false",
			expectNoOp:  true, // Will fail to connect but that's expected
			description: "TLS disabled should attempt insecure connection",
		},
		{
			name:        "TLS enabled",
			tlsEnabled:  "true",
			expectNoOp:  true, // Will fail to connect but that's expected
			description: "TLS enabled should attempt secure connection",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "true")
			os.Setenv("ROX_ML_RISK_SERVICE_ENDPOINT", "localhost:8080")
			os.Setenv("ROX_ML_RISK_SERVICE_TLS", tt.tlsEnabled)
			os.Setenv("ROX_ML_RISK_SERVICE_TIMEOUT", "1s")
			defer func() {
				os.Unsetenv("ROX_ML_RISK_SERVICE_ENABLED")
				os.Unsetenv("ROX_ML_RISK_SERVICE_ENDPOINT")
				os.Unsetenv("ROX_ML_RISK_SERVICE_TLS")
				os.Unsetenv("ROX_ML_RISK_SERVICE_TIMEOUT")
			}()

			Reset()

			client := Singleton()
			assert.NotNil(t, client)

			if tt.expectNoOp {
				_, ok := client.(*noOpClient)
				assert.True(t, ok, "Should fallback to noOpClient when connection fails")
			}
		})
	}
}

func TestSingleton_ConcurrentAccess(t *testing.T) {
	// Reset before test
	Reset()

	const numGoroutines = 10
	clients := make([]MLRiskClient, numGoroutines)
	done := make(chan int, numGoroutines)

	// Access singleton concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			clients[id] = Singleton()
			done <- id
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// All clients should be the same instance
	for i := 1; i < numGoroutines; i++ {
		assert.Same(t, clients[0], clients[i], "All singleton instances should be the same")
	}
}

func TestSingleton_MultipleCallsSameInstance(t *testing.T) {
	Reset()

	client1 := Singleton()
	client2 := Singleton()
	client3 := Singleton()

	assert.Same(t, client1, client2)
	assert.Same(t, client2, client3)
}

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{
			name:     "enabled true",
			envValue: "true",
			expected: true,
		},
		{
			name:     "enabled false",
			envValue: "false",
			expected: false,
		},
		{
			name:     "enabled 1",
			envValue: "1",
			expected: true,
		},
		{
			name:     "enabled 0",
			envValue: "0",
			expected: false,
		},
		{
			name:     "empty value",
			envValue: "",
			expected: false, // Default is false
		},
		{
			name:     "invalid value",
			envValue: "invalid",
			expected: false, // Invalid should default to false
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ROX_ML_RISK_SERVICE_ENABLED", tt.envValue)
			defer os.Unsetenv("ROX_ML_RISK_SERVICE_ENABLED")

			// Reset to ensure we read the new environment value
			Reset()

			result := IsEnabled()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetClientError_BeforeInit(t *testing.T) {
	Reset()

	// GetClientError should initialize singleton if not already done
	err := GetClientError()
	// Error depends on whether ML is enabled and endpoint is reachable
	// But function should not panic
	_ = err // Just ensure no panic
}

func TestReset(t *testing.T) {
	// Initialize singleton first
	client1 := Singleton()
	require.NotNil(t, client1)

	// Reset and get new instance
	Reset()
	client2 := Singleton()
	require.NotNil(t, client2)

	// Should be different instances after reset
	// Note: They might be the same type (noOpClient) but different instances
	// We can't easily test instance equality since they're both noOpClient structs
}

func TestReset_WithNilClient(t *testing.T) {
	// Reset when no client is initialized should not panic
	Reset()
	Reset() // Multiple resets should be safe

	client := Singleton()
	assert.NotNil(t, client)
}

func TestTimeout_Parsing(t *testing.T) {
	tests := []struct {
		name            string
		timeoutValue    string
		expectedDefault bool
	}{
		{
			name:            "valid duration",
			timeoutValue:    "45s",
			expectedDefault: false,
		},
		{
			name:            "valid duration with minutes",
			timeoutValue:    "2m",
			expectedDefault: false,
		},
		{
			name:            "invalid duration",
			timeoutValue:    "invalid",
			expectedDefault: true, // Should use default 30s
		},
		{
			name:            "empty duration",
			timeoutValue:    "",
			expectedDefault: false, // Should use default from env setting
		},
		{
			name:            "negative duration",
			timeoutValue:    "-10s",
			expectedDefault: false, // Negative is technically valid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("ROX_ML_RISK_SERVICE_ENABLED", "true")
			os.Setenv("ROX_ML_RISK_SERVICE_ENDPOINT", "localhost:8080")
			os.Setenv("ROX_ML_RISK_SERVICE_TIMEOUT", tt.timeoutValue)
			defer func() {
				os.Unsetenv("ROX_ML_RISK_SERVICE_ENABLED")
				os.Unsetenv("ROX_ML_RISK_SERVICE_ENDPOINT")
				os.Unsetenv("ROX_ML_RISK_SERVICE_TIMEOUT")
			}()

			Reset()

			// This will attempt to create client and handle timeout parsing
			client := Singleton()
			assert.NotNil(t, client)

			// Should fallback to noOpClient due to connection failure, but parsing should work
			_, ok := client.(*noOpClient)
			assert.True(t, ok)
		})
	}
}

func TestEnvironmentVariableDefaults(t *testing.T) {
	// Clear all ML-related environment variables
	envVars := []string{
		"ROX_ML_RISK_SERVICE_ENABLED",
		"ROX_ML_RISK_SERVICE_ENDPOINT",
		"ROX_ML_RISK_SERVICE_TLS",
		"ROX_ML_RISK_SERVICE_TIMEOUT",
	}

	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}

	Reset()

	// Test defaults
	assert.False(t, IsEnabled()) // Default disabled

	client := Singleton()
	assert.NotNil(t, client)

	// Should be noOpClient due to disabled
	_, ok := client.(*noOpClient)
	assert.True(t, ok)
}

// Benchmark singleton access
func BenchmarkSingleton(b *testing.B) {
	Reset()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Singleton()
	}
}

func BenchmarkIsEnabled(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = IsEnabled()
	}
}

func TestSingleton_StateConsistency(t *testing.T) {
	Reset()

	// Test that singleton state is consistent
	enabled1 := IsEnabled()
	client1 := Singleton()
	enabled2 := IsEnabled()
	client2 := Singleton()
	err1 := GetClientError()
	err2 := GetClientError()

	assert.Equal(t, enabled1, enabled2, "IsEnabled should be consistent")
	assert.Same(t, client1, client2, "Singleton should return same instance")
	assert.Equal(t, err1, err2, "GetClientError should be consistent")
}

func TestSingleton_CloseHandling(t *testing.T) {
	Reset()

	client := Singleton()
	require.NotNil(t, client)

	// Close should not affect singleton state
	err := client.Close()
	assert.NoError(t, err)

	// Getting singleton again should return same instance
	client2 := Singleton()
	assert.Same(t, client, client2)

	// Reset should handle close properly
	Reset() // Should call Close() internally

	// New singleton after reset
	client3 := Singleton()
	assert.NotNil(t, client3)
	// May or may not be same instance depending on implementation
}
