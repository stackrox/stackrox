package flags

import (
	"github.com/stackrox/rox/pkg/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {

	cwd, err := os.Getwd()

	if err != nil {
		t.Logf("Something went wrong: %v", err)
	}

	t.Logf("Current working directory: %v", cwd)

	configFileSet = pointers.Bool(false)
	configEndpointSet = pointers.Bool(false)
	configCaCertFileSet = pointers.Bool(false)
	configApiTokenFileSet = pointers.Bool(false)

	testCases := []struct {
		configFile           string
		expectedProfileName  string
		expectedCaCertPath   string
		expectedApiTokenFile string
		expectedEndpoint     string
		err                  string
	}{
		{
			configFile:           "../../../tests/roxctl/roxctl-instance/test_instance1.yaml",
			expectedProfileName:  "profile-1",
			expectedCaCertPath:   "./deploy/cert",
			expectedApiTokenFile: "REDACTED",
			expectedEndpoint:     "localhost:8000",
		},
		{
			configFile:           "../../../tests/roxctl/roxctl-instance/test_instance2.yaml",
			expectedProfileName:  "dev-environment",
			expectedCaCertPath:   "/etc/ssl/certs",
			expectedApiTokenFile: "/var/secrets/api-token",
			expectedEndpoint:     "localhost:3000",
		},
		{
			configFile:           "../../../tests/roxctl/roxctl-instance/test_instance3.yaml",
			expectedProfileName:  "staging-profile",
			expectedCaCertPath:   "./staging/certs",
			expectedApiTokenFile: "staging-secret-token",
			expectedEndpoint:     "staging.example.com:9000",
		},
		{
			configFile:           "../../../tests/roxctl/roxctl-instance/test_instance4.yaml",
			expectedProfileName:  "production-profile",
			expectedCaCertPath:   "/usr/local/share/ca-certificates",
			expectedApiTokenFile: "/home/user/.secrets/api-token",
			expectedEndpoint:     "https://prod.stackrox.example.com",
		},
		{
			configFile:           "../../../tests/roxctl/roxctl-instance/test_instance5.yaml",
			expectedProfileName:  "test-profile",
			expectedCaCertPath:   "./test/certificates",
			expectedApiTokenFile: "test-token-redacted",
			expectedEndpoint:     "127.0.0.1:7000",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.configFile, func(t *testing.T) {
			instance, err := readConfig(tc.configFile)

			if tc.err == "" {
				require.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.err)
			}

			assert.Equal(t, tc.expectedProfileName, instance.ProfileName)
			assert.Equal(t, tc.expectedCaCertPath, instance.CaCertificatePath)
			assert.Equal(t, tc.expectedApiTokenFile, instance.ApiTokenFilePath)
			assert.Equal(t, tc.expectedEndpoint, instance.Endpoint)
		})
	}
}
