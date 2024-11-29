package flags

import (
<<<<<<< Updated upstream
<<<<<<< Updated upstream
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
=======
=======
>>>>>>> Stashed changes
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stackrox/rox/pkg/env"
	fileutils "github.com/stackrox/rox/pkg/fileutils"
<<<<<<< Updated upstream
>>>>>>> Stashed changes
=======
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
<<<<<<< Updated upstream
	config      Config
=======
	config      *Config
>>>>>>> Stashed changes
=======
	config      *Config
>>>>>>> Stashed changes
	contextName string
}

var (
<<<<<<< Updated upstream
<<<<<<< Updated upstream
	configFile string
	// configApiTokenFileSet      bool
	// configCaCertificateFileSet bool
	// configEndpointSet          bool
	configFileChanged *bool
=======
=======
>>>>>>> Stashed changes
	configFile        string
	configFileChanged *bool
	// configApiTokenFileSet      bool
	// configCaCertificateFileSet bool
	// configEndpointSet          bool
<<<<<<< Updated upstream
>>>>>>> Stashed changes
=======
>>>>>>> Stashed changes
)

// AddConfigurationFile adds --config-file flag to the base command.
func AddConfigurationFile(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&configFile,
		"config-file",
		"",
		"",
<<<<<<< Updated upstream
<<<<<<< Updated upstream
		"Utilize instance-specific metadata defined within a configuration file. "+
=======
		"Utilize instance-specific metadata defined within a configuration file hello. "+
>>>>>>> Stashed changes
=======
		"Utilize instance-specific metadata defined within a configuration file hello. "+
>>>>>>> Stashed changes
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

<<<<<<< Updated upstream
<<<<<<< Updated upstream
func ReadConfigFile(fileName string) (string, error) {}
=======
=======
>>>>>>> Stashed changes
// Load loads a config file from a given path
//   - Load will prioritize the values that are defined within
//     the configuration files over variables defined within the environment
func Load(configPath string) (*ClientConfig, error) {

	config := NewConfig()

	if configPath != "" {
		path := filepath.Dir(configPath)
		filename := filepath.Base(configPath)
		ext := filepath.Ext(configPath)
	}

	exists, err := fileutils.Exists(path)

	if err != nil {
		return config, err
	}

	// Checks for the existing values defined by environment variables

	// NOTE: *Changed indicates that something is different from the default value
	endpoint, plaintext, err := EndpointAndPlaintextSetting()

	if err != nil {
		return config, err
	}

	caCertFile := CAFile()
	apiTokenFile := APITokenFile()

	// if len(data) == 0 {
	// 	return &ClientConfig{config, ""}, nil
	// }

	return &ClientConfig{config, ""}, nil
}

// func ReadConfigFile(fileName string) (string, error) {}
<<<<<<< Updated upstream
>>>>>>> Stashed changes
=======
>>>>>>> Stashed changes
