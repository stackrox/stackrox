package azure

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/require"
)

func TestConfigConversion(t *testing.T) {

	testCases := map[string]struct {
		input       *storage.ImageIntegration
		expectedCfg *storage.AzureConfig
		shouldErr   bool
	}{
		"azure config with workload identity": {
			input: storage.ImageIntegration_builder{
				Type: "azure",
				Azure: storage.AzureConfig_builder{
					Endpoint:   "endpoint",
					WifEnabled: true,
				}.Build(),
			}.Build(),
			expectedCfg: storage.AzureConfig_builder{
				Endpoint:   "endpoint",
				WifEnabled: true,
			}.Build(),
			shouldErr: false,
		},
		"azure config with credentials": {
			input: storage.ImageIntegration_builder{
				Type: "azure",
				Azure: storage.AzureConfig_builder{
					Endpoint: "endpoint",
					Username: "username",
					Password: "password",
				}.Build(),
			}.Build(),
			expectedCfg: storage.AzureConfig_builder{
				Endpoint: "endpoint",
				Username: "username",
				Password: "password",
			}.Build(),
			shouldErr: false,
		},
		"docker config": {
			input: storage.ImageIntegration_builder{
				Type: "azure",
				Docker: storage.DockerConfig_builder{
					Endpoint: "endpoint",
					Username: "username",
					Password: "password",
				}.Build(),
			}.Build(),
			expectedCfg: storage.AzureConfig_builder{
				Endpoint:   "endpoint",
				Username:   "username",
				Password:   "password",
				WifEnabled: false,
			}.Build(),
			shouldErr: false,
		},
		"no valid type": {
			input: storage.ImageIntegration_builder{
				Type: "ecr",
				Ecr: storage.ECRConfig_builder{
					Endpoint: "endpoint",
				}.Build(),
			}.Build(),
			expectedCfg: nil,
			shouldErr:   true,
		},
		"invalid - no endpoint": {
			input: storage.ImageIntegration_builder{
				Type: "azure",
				Azure: storage.AzureConfig_builder{
					Username: "username",
					Password: "password",
				}.Build(),
			}.Build(),
			expectedCfg: nil,
			shouldErr:   true,
		},
		"invalid - no password": {
			input: storage.ImageIntegration_builder{
				Type: "azure",
				Azure: storage.AzureConfig_builder{
					Username: "username",
				}.Build(),
			}.Build(),
			expectedCfg: nil,
			shouldErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			cfg, err := getACRConfig(tc.input)
			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			protoassert.Equal(t, tc.expectedCfg, cfg)
		})
	}
}
