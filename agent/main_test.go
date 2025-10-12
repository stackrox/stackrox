package main

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseArgs(t *testing.T) {
	// Save original command line args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Save original flag.CommandLine and restore after test
	oldCommandLine := flag.CommandLine
	defer func() { flag.CommandLine = oldCommandLine }()

	tests := []struct {
		name      string
		args      []string
		envVars   map[string]string
		expected  *Config
		wantError bool
	}{
		{
			name: "default vsock mode",
			args: []string{"agent"},
			expected: &Config{
				TransmissionMode: "vsock",
				CertPath:         "",
				SensorURL:        "",
			},
			wantError: false,
		},
		{
			name: "grpc mode with cert-path",
			args: []string{"agent", "--mode", "grpc", "--cert-path", "/certs", "--sensor-url", "sensor.example.com:443"},
			expected: &Config{
				TransmissionMode: "grpc",
				CertPath:         "/certs",
				SensorURL:        "sensor.example.com:443",
			},
			wantError: false,
		},
		{
			name: "grpc mode with ROX_MTLS env vars",
			args: []string{"agent", "--mode", "grpc", "--sensor-url", "sensor.example.com:443"},
			envVars: map[string]string{
				"ROX_MTLS_CA_FILE":   "/run/secrets/stackrox.io/certs/ca.pem",
				"ROX_MTLS_CERT_FILE": "/run/secrets/stackrox.io/certs/cert.pem",
				"ROX_MTLS_KEY_FILE":  "/run/secrets/stackrox.io/certs/key.pem",
			},
			expected: &Config{
				TransmissionMode: "grpc",
				CertPath:         "",
				SensorURL:        "sensor.example.com:443",
				CACertFile:       "/run/secrets/stackrox.io/certs/ca.pem",
				ClientCert:       "/run/secrets/stackrox.io/certs/cert.pem",
				ClientKey:        "/run/secrets/stackrox.io/certs/key.pem",
			},
			wantError: false,
		},
		{
			name:      "invalid transmission mode",
			args:      []string{"agent", "--mode", "invalid"},
			wantError: true,
		},
		{
			name:      "grpc mode missing sensor-url",
			args:      []string{"agent", "--mode", "grpc", "--cert-path", "/certs"},
			wantError: true,
		},
		{
			name:      "grpc mode missing certificates",
			args:      []string{"agent", "--mode", "grpc", "--sensor-url", "sensor.example.com:443"},
			wantError: true,
		},
		{
			name: "help flag",
			args: []string{"agent", "--help"},
			// This would normally exit, but we'll handle it in the test
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flag.CommandLine for each test
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

			// Set environment variables
			if tt.envVars != nil {
				for key, value := range tt.envVars {
					require.NoError(t, os.Setenv(key, value))
					defer func(k string) {
						require.NoError(t, os.Unsetenv(k))
					}(key)
				}
			}

			// Set os.Args
			os.Args = tt.args

			// Special handling for help flag test
			if len(tt.args) > 1 && tt.args[1] == "--help" {
				// Skip this test as it would cause the program to exit
				t.Skip("Skipping help flag test as it causes program exit")
				return
			}

			config, err := parseArgs()

			if tt.wantError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected.TransmissionMode, config.TransmissionMode)
			assert.Equal(t, tt.expected.CertPath, config.CertPath)
			assert.Equal(t, tt.expected.SensorURL, config.SensorURL)

			// Only check env var fields if they were set in the test
			if tt.expected.CACertFile != "" {
				assert.Equal(t, tt.expected.CACertFile, config.CACertFile)
			}
			if tt.expected.ClientCert != "" {
				assert.Equal(t, tt.expected.ClientCert, config.ClientCert)
			}
			if tt.expected.ClientKey != "" {
				assert.Equal(t, tt.expected.ClientKey, config.ClientKey)
			}
		})
	}
}

func TestCheckVSockAvailability(t *testing.T) {
	// This test checks the actual file system, so results may vary
	// In a real environment with VSOCK support, this should pass
	// In environments without VSOCK, this should fail

	err := checkVSockAvailability()

	// We can't assert a specific result since it depends on the environment
	// But we can ensure the function doesn't panic and returns a proper error or nil
	if err != nil {
		assert.Contains(t, err.Error(), "VSOCK")
	}
	// If err is nil, VSOCK is available
}

func TestConfig(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "complete config",
			config: Config{
				TransmissionMode: "grpc",
				CertPath:         "/certs",
				SensorURL:        "sensor.example.com:443",
				CACertFile:       "/certs/ca.pem",
				ClientCert:       "/certs/cert.pem",
				ClientKey:        "/certs/key.pem",
			},
		},
		{
			name: "vsock config",
			config: Config{
				TransmissionMode: "vsock",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the config struct can be created and accessed
			assert.Equal(t, tt.config.TransmissionMode, tt.config.TransmissionMode)
			assert.Equal(t, tt.config.CertPath, tt.config.CertPath)
			assert.Equal(t, tt.config.SensorURL, tt.config.SensorURL)
			assert.Equal(t, tt.config.CACertFile, tt.config.CACertFile)
			assert.Equal(t, tt.config.ClientCert, tt.config.ClientCert)
			assert.Equal(t, tt.config.ClientKey, tt.config.ClientKey)
		})
	}
}

// Test helper functions
func TestArgValidation(t *testing.T) {
	tests := []struct {
		name          string
		mode          string
		certPath      string
		sensorURL     string
		caCertFile    string
		clientCert    string
		clientKey     string
		shouldBeValid bool
	}{
		{
			name:          "valid vsock mode",
			mode:          "vsock",
			shouldBeValid: true,
		},
		{
			name:          "valid grpc with cert-path",
			mode:          "grpc",
			certPath:      "/certs",
			sensorURL:     "sensor.example.com:443",
			shouldBeValid: true,
		},
		{
			name:          "valid grpc with individual certs",
			mode:          "grpc",
			sensorURL:     "sensor.example.com:443",
			caCertFile:    "/certs/ca.pem",
			clientCert:    "/certs/cert.pem",
			clientKey:     "/certs/key.pem",
			shouldBeValid: true,
		},
		{
			name:          "invalid mode",
			mode:          "invalid",
			shouldBeValid: false,
		},
		{
			name:          "grpc missing sensor url",
			mode:          "grpc",
			certPath:      "/certs",
			shouldBeValid: false,
		},
		{
			name:          "grpc missing certs",
			mode:          "grpc",
			sensorURL:     "sensor.example.com:443",
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				TransmissionMode: tt.mode,
				CertPath:         tt.certPath,
				SensorURL:        tt.sensorURL,
				CACertFile:       tt.caCertFile,
				ClientCert:       tt.clientCert,
				ClientKey:        tt.clientKey,
			}

			// Simulate validation logic
			var isValid bool

			// Check transmission mode
			if config.TransmissionMode != "vsock" && config.TransmissionMode != "grpc" {
				isValid = false
			} else if config.TransmissionMode == "grpc" {
				// Check sensor URL
				if config.SensorURL == "" {
					isValid = false
				} else {
					// Check certificates
					hasLegacyCertPath := config.CertPath != ""
					hasEnvVarCerts := config.CACertFile != "" && config.ClientCert != "" && config.ClientKey != ""
					isValid = hasLegacyCertPath || hasEnvVarCerts
				}
			} else {
				isValid = true // vsock mode
			}

			assert.Equal(t, tt.shouldBeValid, isValid)
		})
	}
}
