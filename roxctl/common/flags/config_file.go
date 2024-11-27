package flags

import (
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
)

// TODO: Where do we put these type definitions?
type Config struct {
	ApiTokenFile string
	Endpoint     string
	CaCertFile   string
}

func NewConfig() *Config {
	return &Config{
		ApiTokenFile: "",
		Endpoint:     "",
		CaCertFile:   "",
	}

}

type ClientConfig struct {
	config      Config
	contextName string
}

var (
	configFile string
	// configApiTokenFileSet      bool
	// configCaCertificateFileSet bool
	// configEndpointSet          bool
	configFileChanged *bool
)

// AddConfigurationFile adds --config-file flag to the base command.
func AddConfigurationFile(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&configFile,
		"config-file",
		"",
		"",
		"Utilize instance-specific metadata defined within a configuration file. "+
			"Alternatively, set the path via the ROX_CONFIG_FILE environment variable")
	configFileChanged = &c.PersistentFlags().Lookup("config-file").Changed
}

// ConfigurationFile returns the currently specified configuration file name.
func ConfigurationFile() string {
	return flagOrSettingValue(configFile, ConfigurationFileChanged(), env.ConfigFileEnv)
}

// ConfigurationFileChanged returns whether the config-file is provided as an argument.
func ConfigurationFileChanged() bool {
	return configFileChanged != nil && *configFileChanged
}

func ReadConfigFile(fileName string) (string, error) {}
