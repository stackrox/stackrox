package quay

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestValidate(t *testing.T) {
	testutils.RunWithFeatureFlagEnabled(t, features.QuayRobotAccounts, func(t *testing.T) {
		robotAccount := &storage.QuayConfig_RobotAccount{
			Username: "robotUser",
			Password: "robotPassword",
		}
		registryOnly := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY}
		scannerOnly := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER}
		bothIntegrations := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY, storage.ImageIntegrationCategory_SCANNER}

		cases := []struct {
			name         string
			categories   []storage.ImageIntegrationCategory
			shouldError  bool
			endpoint     string
			oauthToken   string
			robotAccount *storage.QuayConfig_RobotAccount
		}{
			// Test missing endpoint
			{name: "Error if no endpoint for registry", categories: registryOnly, shouldError: true},
			{name: "Error if no endpoint for scanner", categories: scannerOnly, shouldError: true},
			{name: "Error if no endpoint for registry & scanner both", categories: bothIntegrations, shouldError: true},

			// Test just registry integration
			{name: "Can skip token and robot creds for registry", categories: registryOnly, shouldError: false, endpoint: "https://quay.io"},
			{name: "Can use just token for registry", categories: registryOnly, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234"},
			{name: "Can use just robot creds for registry", categories: registryOnly, shouldError: false, endpoint: "https://quay.io", robotAccount: robotAccount},
			{name: "Error using both token and robot for registry", categories: registryOnly, shouldError: true, endpoint: "https://quay.io", oauthToken: "abcd$1234", robotAccount: robotAccount},

			// Test just scanner integration
			{name: "Can skip token and robot creds for scanner", categories: scannerOnly, shouldError: false, endpoint: "https://quay.io"},
			{name: "Can use just token for scanner", categories: scannerOnly, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234"},
			{name: "Error using just robot creds for scanner", categories: scannerOnly, shouldError: true, endpoint: "https://quay.io", robotAccount: robotAccount},
			{name: "Error using both token and robot for scanner", categories: scannerOnly, shouldError: true, endpoint: "https://quay.io", oauthToken: "abcd$1234", robotAccount: robotAccount},

			// Test integrating both
			{name: "Can skip token and robot creds for both registry & scanner", categories: bothIntegrations, shouldError: false, endpoint: "https://quay.io"},
			{name: "Can use just token for both registry & scanner", categories: bothIntegrations, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234"},
			{name: "Error using just robot creds for both registry & scanner", categories: bothIntegrations, shouldError: true, endpoint: "https://quay.io", robotAccount: robotAccount},
			{name: "Can use both token and robot for both registry & scanner", categories: bothIntegrations, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234", robotAccount: robotAccount},

			// Test incomplete robot account creds
			{name: "Error if missing username for robot account", categories: registryOnly, shouldError: true, endpoint: "https://quay.io", robotAccount: &storage.QuayConfig_RobotAccount{Password: "password"}},
			{name: "Error if missing password for robot account", categories: registryOnly, shouldError: true, endpoint: "https://quay.io", robotAccount: &storage.QuayConfig_RobotAccount{Username: "password"}},
			{name: "Error if missing username & password for robot account", categories: registryOnly, shouldError: true, endpoint: "https://quay.io", robotAccount: &storage.QuayConfig_RobotAccount{}},
		}

		for _, c := range cases {
			t.Run(c.name, func(t *testing.T) {
				quayConfig := &storage.QuayConfig{
					Endpoint:                 c.endpoint,
					OauthToken:               c.oauthToken,
					RegistryRobotCredentials: c.robotAccount,
				}
				if c.shouldError {
					assert.Error(t, validate(quayConfig, c.categories))
				} else {
					assert.NoError(t, validate(quayConfig, c.categories))
				}
			})
		}
	})
}

// Split from other test for easier reading and deletion
// TODO: Remove when ROX_QUAY_ROBOT_ACCOUNTS is removed.
func TestValidateWithoutFeatureFlag(t *testing.T) {
	t.Setenv(features.QuayRobotAccounts.EnvVar(), "false")

	if features.QuayRobotAccounts.Enabled() {
		t.Skip("Skip test if ROX_QUAY_ROBOT_ACCOUNTS is enabled")
	}

	registryOnly := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY}
	scannerOnly := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER}
	bothIntegrations := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY, storage.ImageIntegrationCategory_SCANNER}

	cases := []struct {
		name         string
		categories   []storage.ImageIntegrationCategory
		shouldError  bool
		endpoint     string
		oauthToken   string
		robotAccount *storage.QuayConfig_RobotAccount
	}{
		// Test missing endpoint
		{name: "Error if no endpoint for registry", categories: registryOnly, shouldError: true},
		{name: "Error if no endpoint for scanner", categories: scannerOnly, shouldError: true},
		{name: "Error if no endpoint for registry & scanner both", categories: bothIntegrations, shouldError: true},

		// Test just registry integration
		{name: "Can skip token for registry", categories: registryOnly, shouldError: false, endpoint: "https://quay.io"},
		{name: "Can use token for registry", categories: registryOnly, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234"},

		// Test just scanner integration
		{name: "Can skip token for scanner", categories: scannerOnly, shouldError: false, endpoint: "https://quay.io"},
		{name: "Can use just token for scanner", categories: scannerOnly, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234"},

		// Test integrating both
		{name: "Can skip token for both registry & scanner", categories: bothIntegrations, shouldError: false, endpoint: "https://quay.io"},
		{name: "Can use just token for both registry & scanner", categories: bothIntegrations, shouldError: false, endpoint: "https://quay.io", oauthToken: "abcd$1234"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			quayConfig := &storage.QuayConfig{
				Endpoint:                 c.endpoint,
				OauthToken:               c.oauthToken,
				RegistryRobotCredentials: c.robotAccount,
			}
			if c.shouldError {
				assert.Error(t, validate(quayConfig, c.categories))
			} else {
				assert.NoError(t, validate(quayConfig, c.categories))
			}
		})
	}
}
