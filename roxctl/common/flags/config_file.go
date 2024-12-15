package flags

import (
	"fmt"

	"os"
	"os/user"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"

	"syscall"
)

type Instance struct {
	InstanceName      string `json:"name"`
	CaCertificatePath string `json:"caCertPath,omitempty"`
	CaCertificate     string `json:"caCert,omitempty"`
	ApiTokenFilePath  string `json:"apiTokenPath,omitempty"`
	ApiToken          string `json:"apiToken,omitempty"`
	Endpoint          string `json:"endpoint,omitempty"`
}

type instanceConfig struct {
	Version  string   `json:"version"`
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

// InlineCaCertificate returns the configuration-defined CA Certificate.
func InlineCaCertificate() string {
	if ConfigurationFileChanged() {
		return config.Instance.CaCertificate
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

// InlineApiToken returns the configuration-defined ApiToken.
func InlineApiToken() string {
	if ConfigurationFileChanged() {
		return config.Instance.ApiToken
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

// checkFilePermissionsAndOwnership is a utility function for checking for 600 file
// permissions and file ownership.
func checkFilePermissionsAndOwnership(path string) error {

	// 1. Obtain file info
	fileInfo, err := os.Stat(path)
	if err != nil {
		return errors.Wrapf(err, "failed to stat file: %v", err)
	}

	// 2. Obtain file permissions information
	fileMode := fileInfo.Mode().Perm()
	if fileMode != 0o600 {
		return errors.Wrapf(err, "file does not have 600 permisisons, got: %v", fileMode)
	}

	// 3. Obtain file ownership info and current user info
	fileOwner, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.Wrapf(err, "failed to get file system info")
	}

	fileOwnerUid := fmt.Sprintf("%d", fileOwner.Uid)

	currentUser, err := user.Current()
	if err != nil {
		return errors.Wrapf(err, "failed to get current user: %v", err)
	}

	if fileOwnerUid != currentUser.Uid {
		return errors.Wrapf(err, "file is not owned by current user, instead: %v", fileOwnerUid)
	}

	return nil
}

// readConfig is a utilty function for reading YAML-based configuration files
func readConfig(path string) (*Instance, error) {
	var conf Instance

	err := checkFilePermissionsAndOwnership(path)

	if err != nil {
		return nil, errors.Wrapf(err, "file permission or ownership error: %v", err)
	}

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
