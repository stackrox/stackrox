package azure

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/require"
)

func TestConfigConversion(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input       *storage.ImageIntegration
		expectedCfg *storage.AzureConfig
		shouldErr   bool
	}{
		"azure config with workload identity": {
			input: &storage.ImageIntegration{
				Type: "azure",
				IntegrationConfig: &storage.ImageIntegration_Azure{
					Azure: &storage.AzureConfig{
						Endpoint:   "endpoint",
						WifEnabled: true,
					},
				},
			},
			expectedCfg: &storage.AzureConfig{
				Endpoint:   "endpoint",
				WifEnabled: true,
			},
			shouldErr: false,
		},
		"azure config with credentials": {
			input: &storage.ImageIntegration{
				Type: "azure",
				IntegrationConfig: &storage.ImageIntegration_Azure{
					Azure: &storage.AzureConfig{
						Endpoint: "endpoint",
						Username: "username",
						Password: "password",
					},
				},
			},
			expectedCfg: &storage.AzureConfig{
				Endpoint: "endpoint",
				Username: "username",
				Password: "password",
			},
			shouldErr: false,
		},
		"docker config": {
			input: &storage.ImageIntegration{
				Type: "azure",
				IntegrationConfig: &storage.ImageIntegration_Docker{
					Docker: &storage.DockerConfig{
						Endpoint: "endpoint",
						Username: "username",
						Password: "password",
					},
				},
			},
			expectedCfg: &storage.AzureConfig{
				Endpoint:   "endpoint",
				Username:   "username",
				Password:   "password",
				WifEnabled: false,
			},
			shouldErr: false,
		},
		"no valid type": {
			input: &storage.ImageIntegration{
				Type: "ecr",
				IntegrationConfig: &storage.ImageIntegration_Ecr{
					Ecr: &storage.ECRConfig{
						Endpoint: "endpoint",
					},
				},
			},
			expectedCfg: nil,
			shouldErr:   true,
		},
		"invalid - no endpoint": {
			input: &storage.ImageIntegration{
				Type: "azure",
				IntegrationConfig: &storage.ImageIntegration_Azure{
					Azure: &storage.AzureConfig{
						Username: "username",
						Password: "password",
					},
				},
			},
			expectedCfg: nil,
			shouldErr:   true,
		},
		"invalid - no password": {
			input: &storage.ImageIntegration{
				Type: "azure",
				IntegrationConfig: &storage.ImageIntegration_Azure{
					Azure: &storage.AzureConfig{
						Username: "username",
					},
				},
			},
			expectedCfg: nil,
			shouldErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
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
