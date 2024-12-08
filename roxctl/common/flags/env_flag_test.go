package flags

import (
	// "fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestFlagOrSettingValue(t *testing.T) {
	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := &cobra.Command{}

	AddPassword(cmd)

	assert.Empty(t, Password())

	// 2. Change the flag value. The changed flag value should be returned, irrespective of whether the setting is set.
	t.Setenv("ROX_ADMIN_PASSWORD", "some-test-value")
	cmd = &cobra.Command{}
	AddPassword(cmd)
	err := cmd.PersistentFlags().Set("password", "some-other-test-value")
	assert.NoError(t, err)
	assert.Equal(t, "some-other-test-value", Password())

	// 3. Default flag value and setting's value set should return the settings value instead.
	t.Setenv("ROX_ADMIN_PASSWORD", "some-test-value")
	cmd = &cobra.Command{}
	AddPassword(cmd)
	assert.Equal(t, "some-test-value", Password())
}

// TODO: How do I make this work from running "test ."?
func TestFlagOrConfigurationValueFlags(t *testing.T) {

	var (
		testFile1 = "./testdata/test_instance1.yaml"
	)
	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := &cobra.Command{
		PersistentPreRunE: LoadConfig,
	}

	AddAPITokenFile(cmd)
	AddConnectionFlags(cmd)
	AddConfigurationFile(cmd)

	assert.False(t, ConfigurationFileChanged())
	assert.False(t, *caCertFileSet)
	assert.Empty(t, *apiTokenFileChanged)
	assert.False(t, *endpointChanged)

	err := cmd.PersistentFlags().Set("config-file", testFile1)
	assert.NoError(t, err)
	assert.Equal(t, testFile1, ConfigurationFileName())

	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.True(t, ConfigurationFileChanged())

	// Execute the command to trigger PersistentPreRunE
	assert.NoError(t, err, "Command execution should not produce an error")

	// 3. Validate that PersistentPreRunE (LoadConfig) ran successfully
	assert.NotNil(t, config, "Config should be initialized after LoadConfig runs")

	// 4. Change flag values. The changed flag values should be returned, irrespective if set by configuration.
	err2 := cmd.PersistentFlags().Set("token-file", "some-test-value")
	assert.NotEmpty(t, config)
	assert.NoError(t, err2)
	assert.Equal(t, "some-test-value", APITokenFile())

	cmd.PersistentFlags().Set("endpoint", "some-other-test-value")
	assert.Equal(t, "some-other-test-value", endpoint)

	cmd.PersistentFlags().Set("ca", "some-other-other-test-value")
	assert.Equal(t, "some-other-other-test-value", caCertFile)

}

func TestFlagOrConfigurationValueEnv(t *testing.T) {
	var (
		testFile1 = "./testdata/test_instance1.yaml"
	)

	cmd := &cobra.Command{
		PersistentPreRunE: LoadConfig,
	}

	AddAPITokenFile(cmd)
	AddConnectionFlags(cmd)
	AddConfigurationFile(cmd)

	assert.False(t, ConfigurationFileChanged())
	assert.False(t, *caCertFileSet)
	assert.Empty(t, *apiTokenFileChanged)
	assert.False(t, *endpointChanged)

	err := cmd.PersistentFlags().Set("config-file", testFile1)
	assert.NoError(t, err, "Setting a configuration file should not produce an error")
	assert.Equal(t, testFile1, ConfigurationFileName())

	// Execute the command to trigger PersistentPreRunE
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NoError(t, err, "Command execution should not produce an error")
	assert.True(t, ConfigurationFileChanged())
	assert.NotEmpty(t, config)

	// 3. Validate that PersistentPreRunE (LoadConfig) ran successfully
	assert.NotNil(t, config, "Config should be initialized after LoadConfig runs")

	// 4. Change environment values. The configuration-defined flag values should be returned, irrespective if set by environment.
	t.Setenv("ROX_API_TOKEN_FILE", "some-other-test-env-value")
	assert.Equal(t, "REDACTED", APITokenFile())

	t.Setenv("ROX_ENDPOINT", "some-test-env-value")
	assert.Equal(t, "localhost:8000", endpoint)

	t.Setenv("ROX_CA_CERT_FILE", "some-other-other-test-env-value")
	assert.Equal(t, "./deploy/cert", caCertFile)

}

func TestFlagOrConfigurationValueFlagsAndEnv(t *testing.T) {
	var (
		testFile1 = "./testdata/test_instance1.yaml"
	)

	cmd := &cobra.Command{
		PersistentPreRunE: LoadConfig,
	}

	AddAPITokenFile(cmd)
	AddConnectionFlags(cmd)
	AddConfigurationFile(cmd)

	assert.False(t, ConfigurationFileChanged())
	assert.False(t, *caCertFileSet)
	assert.Empty(t, *apiTokenFileChanged)
	assert.False(t, *endpointChanged)

	err := cmd.PersistentFlags().Set("config-file", testFile1)
	assert.NoError(t, err, "Setting a configuration file should not produce an error")
	assert.Equal(t, testFile1, ConfigurationFileName())

	// Execute the command to trigger PersistentPreRunE
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NoError(t, err, "Command execution should not produce an error")
	assert.True(t, ConfigurationFileChanged())
	assert.NotEmpty(t, config)

	// 3. Validate that PersistentPreRunE (LoadConfig) ran successfully
	assert.NotNil(t, config, "Config should be initialized after LoadConfig runs")

	// 4. Change environment values.
	t.Setenv("ROX_API_TOKEN_FILE", "some-other-test-env-value")
	t.Setenv("ROX_ENDPOINT", "some-test-env-value")
	t.Setenv("ROX_CA_CERT_FILE", "some-other-other-test-env-value")

	// 5. Change flag values. Flag values should be returned, irrespective of configuration or environment-defined values.
	err = cmd.PersistentFlags().Set("token-file", "some-test-value")
	assert.NoError(t, err, "Setting a configuration file should not produce an error")
	assert.Equal(t, "some-test-value", APITokenFile())

	cmd.PersistentFlags().Set("endpoint", "some-other-test-value")
	assert.Equal(t, "some-other-test-value", endpoint)

	cmd.PersistentFlags().Set("ca", "some-other-other-test-value")
	assert.Equal(t, "some-other-other-test-value", caCertFile)
}

func TestBooleanFlagOrSettingValue(t *testing.T) {
	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := &cobra.Command{}

	AddConnectionFlags(cmd)

	assert.False(t, UseInsecure())

	// 2. Change the flag value. The changed flag value should be returned, irrespective of whether the setting is set.
	t.Setenv("ROX_INSECURE_CLIENT", "true")
	cmd = &cobra.Command{}
	AddConnectionFlags(cmd)
	err := cmd.PersistentFlags().Set("insecure", "false")
	assert.NoError(t, err)
	assert.Equal(t, false, UseInsecure())

	// 3. Default flag value and setting's value set should return the settings value instead.
	t.Setenv("ROX_INSECURE_CLIENT", "true")
	cmd = &cobra.Command{}
	AddConnectionFlags(cmd)
	assert.Equal(t, true, UseInsecure())
}
