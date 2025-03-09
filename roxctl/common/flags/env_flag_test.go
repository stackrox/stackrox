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

// createTestCommand
func createTestCommand(t *testing.T) *cobra.Command {
	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := &cobra.Command{
		PersistentPreRunE: LoadConfig,
	}

	AddAPITokenFile(cmd)
	AddConnectionFlags(cmd)
	AddConfigurationFile(cmd)

	assert.False(t, ConfigurationFileChanged())
	assert.False(t, *caCertFileSet)
	assert.False(t, *apiTokenFileChanged)
	assert.False(t, *endpointChanged)

	return cmd
}

// TestPrecedenceConfigVersusFlags tests the flagOrConfigurationValue by first
// setting a configuration file, then sets flag values. The flag values should
// be returned.
// REMARK: flag > env > config
func TestPrecedenceConfigVersusFlags(t *testing.T) {

	testFile1 := "./testdata/test_instance1.yaml"

	// 1. Default, unchanged flag value and setting not set should lead to the
	// default value being returned.
	cmd := createTestCommand(t)

	err := cmd.PersistentFlags().Set("config-file", testFile1)
	assert.NoError(t, err)
	assert.Equal(t, testFile1, ConfigurationFileName())

	// 2. Execute the command to trigger PersistentPreRunE
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.True(t, ConfigurationFileChanged())
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

// TestPrecedenceConfigVersusEnv tests the flagOrConfigurationValue by first
// setting a configuration file, then sets environment values. The config values should
// be returned.
// REMARK: flag > env > config
func TestPrecedenceConfigVersusEnv(t *testing.T) {

	testFile1 := "./testdata/test_instance1.yaml"

	// 1. Default, unchanged flag value and setting not set should lead to the
	// default value being returned.
	cmd := createTestCommand(t)

	err := cmd.PersistentFlags().Set("config-file", testFile1)
	assert.NoError(t, err, "Setting a configuration file should not produce an error")
	assert.Equal(t, testFile1, ConfigurationFileName())

	// 2. Execute the command to trigger PersistentPreRunE
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

// TestPrecedenceConfigVersusFlagsAndEnv tests the flagOrConfigurationValue by first
// setting a configuration file, then sets environment values and flag values.
// The flag values should be returned.
// REMARK: flag > env > config
func TestPrecedenceConfigVersusFlagsAndEnv(t *testing.T) {

	testFile1 := "./testdata/test_instance1.yaml"

	// 1. Default, unchanged flag value and setting not set should lead to the
	// default value being returned.
	cmd := createTestCommand(t)

	err := cmd.PersistentFlags().Set("config-file", testFile1)
	assert.NoError(t, err, "Setting a configuration file should not produce an error")
	assert.Equal(t, testFile1, ConfigurationFileName())

	// 2. Execute the command to trigger PersistentPreRunE
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

// TestFlagOrConfigurationValueWithFilepathOption tests the function
// flagOrConfigurationValueWithFilepathOption, checking that whenever a YAML is provided with
// both an inline value and a filepath value for a configuration option, the inline value is prioritized.
// inline > filepath
func TestFlagOrConfigurationValueWithFilepathOption(t *testing.T) {

	testFile := "./testdata/test_config6.yaml"

	// 1. Default, unchanged flag value and setting not set should lead to the default value being returned.
	cmd := createTestCommand(t)

	err := cmd.PersistentFlags().Set("config-file", testFile)
	assert.NoError(t, err)
	assert.Equal(t, testFile, ConfigurationFileName())

	// 2. Execute the command to trigger PersistentPreRunE
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.True(t, ConfigurationFileChanged())
	assert.NoError(t, err, "Command execution with valid YAML file should not produce error.")

	// 3. Validate that PersistentPreRunE (LoadConfig) ran successfully
	assert.NotNil(t, config, "Config should be initialized after LoadConfig() runs")

	// 4. Validate that an Instance was added, a CA Certificate filepath was provided, and an inline CA Certificate was added
	assert.True(t, *configInstancesSet, "An instance should have been added.")
	assert.NotNil(t, ConfigCaCertificatePath(), "A CA Certificate filepath should have been provided.")
	assert.True(t, *configCaCertificatePathSet, "A CA Certificate filepath should have been provided.")

	assert.NotNil(t, ConfigInlineCaCertificate(), "An inline CA Certificate should have been provided.")
	assert.True(t, *configInlineCaCertificateSet, "An inline CA Certificate should have been provided.")

	// 5. Validate that an AuthInfo was added, an API Token filepath was provided, and an inline API Token was provided.
	assert.True(t, *configAuthInfoSet, "AuthInfo should have been added.")
	assert.NotNil(t, ConfigApiTokenFilePath(), "An API Token filepath should have been provided.")
	assert.True(t, *configApiTokenFilePathSet, "An API Token filepath should have been provided.")

	assert.NotNil(t, ConfigInlineApiToken(), "An inline API Token should have been provided")
	assert.True(t, *configInlineApiTokenSet, "An inline API Token should have been set.")

	// 5. Determine that inline values should be prioritized
	assert.Equal(t, ConfigInlineApiToken(), APITokenFile(), "The inline API Token configuration value should have been prioritized")
	assert.Equal(t, ConfigInlineCaCertificate(), CAFile(), "The inline API Token configuration value should have been prioritized")
}

// TestBooleanFlagOrConfigurationValue tests the function
// booleanFlagOrConfigurationValue, checking that whenever a YAML boolean flag is provided,
// flag > config > env
func TestBooleanFlagOrConfigurationValue(t *testing.T) {

	testFile := "./testdata/test_config2.yaml"

	// 1. Default, unchanged flag value and setting not set should lead to the
	// default value being returned.
	cmd := createTestCommand(t)

	err := cmd.PersistentFlags().Set("config-file", testFile)
	assert.NoError(t, err)
	assert.Equal(t, testFile, ConfigurationFileName())

	// 2. Execute the command to trigger PersistentPreRunE.
	err = cmd.PersistentPreRunE(cmd, []string{})
	assert.NoError(t, err, "Command execution should not produce an error.")
	assert.True(t, ConfigurationFileChanged())
	assert.NotEmpty(t, config)

	// 3. Show that the configuration values have been set.

	assert.True(t, ConfigurationInstancesChanged(), "Instances should have been added.")
	assert.NotNil(t, ConfigUseDirectGRPC(), "Value should have been set.")
	assert.True(t, *configDirectGRPCSet, "Value should have been set (ie, not the default value of false).")
	assert.True(t, ConfigUseDirectGRPC(), "Configuration value should be true.")

	assert.NotNil(t, ConfigForceHTTP1(), "Value should have been set.")
	assert.False(t, *configForceHTTP1Set, "Value should have not been set (ie, the default value of false).")
	assert.False(t, ConfigForceHTTP1(), "Configuration value should be false.")

	assert.NotNil(t, ConfigUseInsecure(), "Value should have been set.")
	assert.False(t, *configUseInsecureSet, "Value should have not been set (ie, the default value of false).")
	assert.False(t, ConfigUseInsecure(), "Configuration value should be false.")

	assert.NotNil(t, ConfigSkipTLSValidation(), "Value should have been set.")
	assert.True(t, *configInsecureSkipTLSVerifySet, "Value should have been set (ie, not the default value of false).")
	assert.True(t, ConfigSkipTLSValidation(), "Configuration value should be true.")

	// 4. Change flag values (using opposite of configuration files for clarity)
	// and show that the returned value must be from flags, regardless of configuration
	// value
	err = cmd.PersistentFlags().Set("direct-grpc", "false")
	assert.NoError(t, err, "Setting a flag should not produce an error.")
	assert.False(t, directGRPC, "Setting should be false.")

	err = cmd.PersistentFlags().Set("force-http1", "true")
	assert.NoError(t, err, "Setting a flag should not produce an error.")
	assert.True(t, ForceHTTP1(), "Setting should be true.")

	err = cmd.PersistentFlags().Set("insecure", "true")
	assert.NoError(t, err, "Setting a flag should not produce an error.")
	assert.True(t, UseInsecure(), "Setting should be true.")

	err = cmd.PersistentFlags().Set("insecure-skip-tls-verify", "false")
	assert.NoError(t, err, "Setting a flag should not produce an error.")
	assert.False(t, *SkipTLSValidation(), "Setting should be false.")
}
