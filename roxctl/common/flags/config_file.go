package flags

import (
	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"os"

	"k8s.io/utils/pointer"
)

type Instance struct {
	ProfileName       string `json:"name"`
	CaCertificatePath string `json:"caCertPath"`
	ApiTokenFilePath  string `json:"apiTokenPath"`
	Endpoint          string `json:"endpoint"`
}

type yamlConfig interface {
	instanceConfig
}

type instanceConfig struct {
	Instance Instance `yaml:"instance"`
}

var (
	configFile        string
	configFileChanged *bool
	config            *instanceConfig

	configEndpointSet     = pointer.Bool(false)
	configCaCertFileSet   = pointer.Bool(false)
	configApiTokenFileSet = pointer.Bool(false)

	log = logging.CreateLogger(logging.CurrentModule(), 0)
)

// AddConfigurationFile adds --config-file flag to the base command.
func AddConfigurationFile(c *cobra.Command) {
	c.PersistentFlags().StringVarP(&configFile,
		"config-file",
		"C",
		"",
		"Utilize instance-specific metadata defined within a configuration file. "+
			"Alternatively, set the path via the ROX_CONFIG_FILE environment variable")
	configFileChanged = &c.PersistentFlags().Lookup("config-file").Changed
}

// ConfigurationFile returns the currently specified configuration file name.
func ConfigurationFile() string {
	return flagOrSettingValue(configFile, ConfigurationFileChanged(), env.ConfigFileEnv)
}

// ConfigurationFileChanged returns whether the configuration file is provided as an argument.
func ConfigurationFileChanged() bool {
	return configFileChanged != nil && *configFileChanged
}

// CaCertificatePath returns the configuration-defined CA Certificate path.
func (c *instanceConfig) CaCertificatePath() string {
	return c.Instance.CaCertificatePath
}

// ApiTokenFilePath returns the configuration-defined API Token file path.
func (c *instanceConfig) ApiTokenFilePath() string {
	return c.Instance.ApiTokenFilePath
}

// Endpoint returns the configuration-defined endpoint.
func (c *instanceConfig) Endpoint() string {
	return c.Instance.Endpoint
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
func LoadConfig(cmd *cobra.Command, args []string) error {

	if configFile == "" || ConfigurationFileChanged() == false {
		return nil
	}

	instance, err := readConfig(configFile)

	if err != nil || instance == nil {
		config = nil
		log.Errorf("Error reading instance config file: %v", err)
		return err
	}

	config = &instanceConfig{Instance: *instance}

	// CHORE: Remove prints
	// fmt.Println(config)

	// TODO: Fix priority
	// TODO: Should it be file > flag > env?

	if instance.Endpoint != "" {
		endpoint = instance.Endpoint
		configEndpointSet = pointer.Bool(true)
	}

	if instance.CaCertificatePath != "" {
		caCertFile = instance.CaCertificatePath
		configEndpointSet = pointer.Bool(true)
	}

	if instance.ApiTokenFilePath != "" {
		apiTokenFile = instance.ApiTokenFilePath
		configEndpointSet = pointer.Bool(true)
	}

	return nil
}
