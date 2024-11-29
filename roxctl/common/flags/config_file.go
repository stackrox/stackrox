package flags

import (
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"os"
)

type Instance struct {
	ProfileName       string `yaml:"name"`
	CaCertificatePath string `yaml:"caCertPath"`
	ApiTokenFilePath  string `yaml:"apiTokenPath"`
	Endpoint          string `yaml:"endpoint"`
}

type yamlConfig interface {
	instanceConfig
}

type instanceConfig struct {
	Instance Instance `yaml:"instance"`
}

var (
	configFile string
	// configApiTokenFileSet      bool
	// configCaCertificateFileSet bool
	// configEndpointSet          bool
	configFileChanged *bool

	log = logging.CreateLogger(logging.CurrentModule(), 0)
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

// readConfig is a utilty function for reading YAML-based configuration files
func readConfig(path string) (*Instance, error) {
	var conf Instance
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &conf, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(bytes, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}

// Load loads a config file from a given path
//   - Load will prioritize the values that are defined within
//     the configuration files over variables defined within the environment
func Load() (*instanceConfig, error) {

	var config *instanceConfig

	if configFile == "" || ConfigurationFileChanged() == false {
		return nil, nil
	}

	instance, err := readConfig(configFile)

	if err != nil {
		config = nil
		log.Errorf("Error reading instance config file: %v", err)
	}

	config.Instance = *instance

	// TODO: Fix priority
	// TODO: Should it be file > flag > env?

	if instance.Endpoint != "" {
		endpoint = instance.Endpoint
	}

	if instance.CaCertificatePath != "" {
		caCertFile = instance.CaCertificatePath
	}

	if instance.ApiTokenFilePath != "" {
		apiTokenFile = instance.ApiTokenFilePath
	}

	return config, nil
}
