package flags

import (
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEndpointAndPlaintextSetting(t *testing.T) {
	endpointChanged = pointers.Bool(false)
	plaintextSet = pointers.Bool(false)

	testCases := []struct {
		givenEndpoint    string
		expectedEndpoint string
		usePlaintext     bool
		err              string
	}{
		{
			givenEndpoint:    "localhost:8443",
			expectedEndpoint: "localhost:8443",
		},
		{
			givenEndpoint:    "https://localhost",
			expectedEndpoint: "localhost:443",
		},
		{
			givenEndpoint:    "https://example.com:443",
			expectedEndpoint: "example.com:443",
		},
		{
			givenEndpoint:    "http://example.com:80",
			expectedEndpoint: "example.com:80",
			usePlaintext:     true,
		},
		{
			givenEndpoint:    "https://example.com",
			expectedEndpoint: "example.com:443",
		},
		{
			givenEndpoint:    "http://example.com",
			expectedEndpoint: "example.com:80",
			usePlaintext:     true,
		},
		{
			givenEndpoint:    "http://128.66.0.1",
			expectedEndpoint: "128.66.0.1:80",
			usePlaintext:     true,
		},
		{
			givenEndpoint: "128.66.0.1",
			err:           "invalid arguments: address 128.66.0.1: missing port in address",
		},
		{
			givenEndpoint: "example.com",
			err:           "invalid arguments: address example.com: missing port in address",
		},
		{
			givenEndpoint: "example.com:80:80",
			err:           "invalid arguments: address example.com:80:80: too many colons in address",
		},
		{
			givenEndpoint: "https://host:port",
			err:           `invalid arguments: parse "https://host:port": invalid port ":port" after host`,
		},
		{
			// SplitHostPort does not verify if port is numeric (but url.Parse does).
			givenEndpoint:    "host:port",
			expectedEndpoint: "host:port",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.givenEndpoint, func(t *testing.T) {
			t.Setenv(env.EndpointEnv.EnvVar(), tc.givenEndpoint)
			host, usePlaintext, err := EndpointAndPlaintextSetting()
			if tc.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}
			assert.Equal(t, tc.expectedEndpoint, host)
			assert.Equal(t, tc.usePlaintext, usePlaintext)
		})
	}
}

func TestEndpointWasExplicitlyProvided(t *testing.T) {
	testCases := []struct {
		name                string
		endpointFlag        bool
		envVarValue         string
		kubeContextFlag     bool
		kubeContextEnvValue string
		expectedExplicit    bool
	}{
		{
			name:             "default endpoint - not explicit",
			endpointFlag:     false,
			envVarValue:      "",
			kubeContextFlag:  false,
			expectedExplicit: false,
		},
		{
			name:             "endpoint flag set - explicit",
			endpointFlag:     true,
			envVarValue:      "",
			kubeContextFlag:  false,
			expectedExplicit: true,
		},
		{
			name:             "env var set - explicit",
			endpointFlag:     false,
			envVarValue:      "central.example.com:443",
			kubeContextFlag:  false,
			expectedExplicit: true,
		},
		{
			name:             "kube context flag enabled - explicit",
			endpointFlag:     false,
			envVarValue:      "",
			kubeContextFlag:  true,
			expectedExplicit: true,
		},
		{
			name:                "kube context env var enabled - explicit",
			endpointFlag:        false,
			envVarValue:         "",
			kubeContextFlag:     false,
			kubeContextEnvValue: "true",
			expectedExplicit:    true,
		},
		{
			name:             "endpoint flag and env var both set - explicit",
			endpointFlag:     true,
			envVarValue:      "central.example.com:443",
			kubeContextFlag:  false,
			expectedExplicit: true,
		},
		{
			name:             "all three set - explicit",
			endpointFlag:     true,
			envVarValue:      "central.example.com:443",
			kubeContextFlag:  true,
			expectedExplicit: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset state
			endpointChanged = pointers.Bool(tc.endpointFlag)
			useKubeContext = tc.kubeContextFlag

			// Clear and optionally set environment variables
			t.Setenv(env.EndpointEnv.EnvVar(), tc.envVarValue)
			if tc.kubeContextEnvValue != "" {
				t.Setenv(env.UseCurrentKubeContext.EnvVar(), tc.kubeContextEnvValue)
			} else {
				t.Setenv(env.UseCurrentKubeContext.EnvVar(), "")
			}

			result := EndpointWasExplicitlyProvided()
			assert.Equal(t, tc.expectedExplicit, result,
				"EndpointWasExplicitlyProvided() returned %v, expected %v", result, tc.expectedExplicit)
		})
	}
}
