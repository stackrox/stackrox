package flags

import (
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
)

type Instance struct {
	ProfileName       string `json:"name"`
	CaCertificatePath string `json:"caCertPath"`
	ApiTokenFilePath  string `json:"apiTokenPath"`
	Endpoint          string `json:"endpoint"`
}

type instanceConfig struct {
	Instance Instance `yaml:"instance"`
}

var (
	configFile    string
	configFileSet = pointers.Bool(false)
	config        *instanceConfig

	configEndpointSet     = pointers.Bool(false)
	configCaCertFileSet   = pointers.Bool(false)
	configApiTokenFileSet = pointers.Bool(false)

	log = logging.LoggerForModule()
)

// AddConfigurationFile adds --config-file flag to the base command.
func AddConfigurationFile(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&configFile,
		"config-file",
		"C",
		"",
		"Utilize instance-specific metadata defined within a configuration file. "+
			"Alternatively, set the path via the ROX_CONFIG_FILE environment variable")
	configFileSet = &c.PersistentFlags().Lookup("config-file").Changed
}

// ConfigurationFile returns the currently specified configuration file name.
func ConfigurationFileName() string {
	return flagOrSettingValue(configFile, ConfigurationFileChanged(), env.ConfigFileEnv)
}

// ConfigurationFileChanged returns whether the configuration file is provided as an argument.
func ConfigurationFileChanged() bool {
	return configFileSet != nil && *configFileSet
}

// CaCertificatePath returns the configuration-defined CA Certificate path.
func CaCertificatePath() string {
	if ConfigurationFileChanged() {
		return config.Instance.CaCertificatePath
	}

	return ""
}

// ApiTokenFilePath returns the configuration-defined API Token file path.
func ApiTokenFilePath() string {
	if ConfigurationFileChanged() {
		return config.Instance.ApiTokenFilePath
	}

	return ""
}

// Endpoint returns the configuration-defined endpoint.
func Endpoint() string {
	if ConfigurationFileChanged() {
		return config.Instance.Endpoint
	}

	return ""
}

// readConfig is a utilty function for reading YAML-based configuration files
func readConfig(path string) (*Instance, error) {
	var conf Instance
	bytes, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &conf, nil
		}
		return nil, errors.Wrapf(err, "failed to retrieve config from file %q", path)
	}
	if err := yaml.Unmarshal(bytes, &conf); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve config from file %q", path)
	}

	return &conf, nil
}

// Load loads a config file from a given path
//   - Load will prioritize the values that are defined within
//     the configuration files over variables defined within the environment
func LoadConfig(cmd *cobra.Command, args []string) error {

	if configFile == "" || !ConfigurationFileChanged() {
		return nil
	}

	instance, err := readConfig(configFile)

	if err != nil || instance == nil {
		config = nil
		log.Errorf("Error reading instance config file: %v", err)
		return err
	}

	config = &instanceConfig{Instance: *instance}

	// TODO: Should it be file > flag > env?

	if instance.Endpoint != "" {
		endpoint = instance.Endpoint
		configEndpointSet = pointers.Bool(true)
	}

	if instance.CaCertificatePath != "" {
		caCertFile = instance.CaCertificatePath
		configCaCertFileSet = pointers.Bool(true)
	}

	if instance.ApiTokenFilePath != "" {
		apiTokenFile = instance.ApiTokenFilePath
		configApiTokenFileSet = pointers.Bool(true)
	}

	return nil
}
