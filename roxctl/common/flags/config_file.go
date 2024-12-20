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
	InstanceName          string `json:"name"` // TODO(2): Is this the same as flags.endpoint.ServerName?
	Endpoint              string `json:"endpoint,omitempty"`
	CaCertificatePath     string `json:"caCertPath,omitempty"`
	CaCertificate         string `json:"caCert,omitempty"`
	Plaintext             bool   `json:"plaintext,omitempty"`
	DirectGRPC            bool   `json:"directGrpc,omitempty"`
	ForceHTTP1            bool   `json:"forceHttp,omitempty"`
	Insecure              bool   `json:"insecure,omitempty"`
	InsecureSkipTLSVerify bool   `json:"insecureSkipTLSVerify,omitempty"`
}

// Context is a tuple of references to a Stackrox instance (ie, how I communicate with a Stackrox
// instance) and a user (ie, how I identify myself)
type Context struct {
	Name     string `json:"name"`
	Instance string `json:"instance"`
	AuthInfo string `json:"user,omitempty"`
}

// AuthInfo contains information that describes identity information. This is used to tell the
// Stackrox instance who you are.
type AuthInfo struct {
	// Username is the username for basic authentication to the Stackrox instance.
	// +optional
	Username string `json:"username"`
	// Password is the password for basic authentication to the Stackrox instance.
	// +optional
	Password string `json:"password,omitempty"`
	// ApiToken is an inline bearer token for authentication to the Stackrox instance.
	// +optional
	ApiToken string `json:"apiToken,omitempty"`
	// ApiTokenFilePath is a pointer to a file that contains a bearer token (as described above).
	// If both ApiToken and ApiTokenFilePath are present, then ApiToken takes precedence.
	// +optional
	ApiTokenFilePath string `json:"apiTokenPath,omitempty"` //
}

// InstanceConfig holds the information needed to connect to remote Stackrox instances as a
// given user.
// TODO: Enable users to store multiple instances, credentials, and contexts (maps?).
type InstanceConfig struct {
	Version     string    `json:"version"`
	Instances   *Instance `json:"instances"`
	AuthInfo    *AuthInfo `json:"users"`
	Contexts    *Context  `json:"contexts"`        // TODO: Add context functionality
	CurrContext string    `json:"current-context"` // TODO: Add current context functionality
}

var (
	configFile    string
	configFileSet = pointers.Bool(false)
	config        *InstanceConfig

	// Flags related to Instance struct
	configInstancesSet             = pointers.Bool(false) // Existence flag
	configEndpointSet              = pointers.Bool(false)
	configCaCertificatePathSet     = pointers.Bool(false)
	configInlineCaCertificateSet   = pointers.Bool(false)
	configPlaintextSet             = pointers.Bool(false)
	configDirectGRPCSet            = pointers.Bool(false)
	configForceHTTP1Set            = pointers.Bool(false)
	configUseInsecureSet           = pointers.Bool(false)
	configInsecureSkipTLSVerifySet = pointers.Bool(false)

	// Flags related to AuthInfo struct
	configAuthInfoSet         = pointers.Bool(false) // Existence flag
	configUsernameSet         = pointers.Bool(false)
	configPasswordSet         = pointers.Bool(false)
	configApiTokenFilePathSet = pointers.Bool(false)
	configInlineApiTokenSet   = pointers.Bool(false)

	// Flags related to Context struct
	configContextsSet         = pointers.Bool(false) // Existence flag
	configContextInstanceSet  = pointers.Bool(false)
	configContextsAuthInfoSet = pointers.Bool(false)

	// Flags related to CurrContext
	configCurrContextSet = pointers.Bool(false)

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

func ConfigurationInstancesChanged() bool {
	return ConfigurationFileChanged() && (configInstancesSet != nil && *configInstancesSet)
}

func ConfigurationAuthInfoChanged() bool {
	return ConfigurationFileChanged() && (configAuthInfoSet != nil && *configAuthInfoSet)
}

func ConfigurationContextsChanged() bool {
	return ConfigurationFileChanged() && (configContextsSet != nil && *configContextsSet)
}

func ConfigurationCurrContextChanged() bool {
	return ConfigurationFileChanged() && (configCurrContextSet != nil && *configCurrContextSet)
}

// Endpoint returns the configuration-defined endpoint.
func ConfigEndpoint() string {
	if ConfigurationInstancesChanged() {
		return config.Instances.Endpoint
	}

	return ""
}

// CaCertificatePath returns the configuration-defined CA Certificate path.
func ConfigCaCertificatePath() string {
	if ConfigurationInstancesChanged() {
		return config.Instances.CaCertificatePath
	}

	return ""
}

// CaCertificate returns the configuration-defined inline CA Certificate.
func ConfigInlineCaCertificate() string {
	if ConfigurationInstancesChanged() {
		return config.Instances.CaCertificate
	}

	return ""
}

// Plaintext returns the configuration-defined plaintext.
func ConfigPlaintext() string {
	if ConfigurationInstancesChanged() {
		return config.Instances.Plaintext
	}

	return ""
}

// DirectGRPC returns the configuration-defined Direct GRPC option.
func ConfigUseDirectGRPC() bool {
	if ConfigurationInstancesChanged() {
		return config.Instances.DirectGRPC
	}

	return false // default value
}

// ForceHTTP1 returns the configuration-defined Force HTTP option.
func ConfigForceHTTP1() bool {
	if ConfigurationInstancesChanged() {
		return config.Instances.ForceHTTP1
	}

	return false
}

// ConfigUseInsecure returns the configuration-defined Insecure option.
func ConfigUseInsecure() bool {
	if ConfigurationInstancesChanged() {
		return config.Instances.Insecure
	}

	return false
}

// InsecureSkipTLSVerify returns the configuration-defined Insecure Skip TLS Verify option.
func ConfigSkipTLSValidation() bool {
	if ConfigurationInstancesChanged() {
		return config.Instances.InsecureSkipTLSVerify
	}

	return false
}

// Username returns the configuration-defined username.
func ConfigUsername() string {
	if ConfigurationAuthInfoChanged() {
		return config.AuthInfo.Username
	}

	return ""
}

// Password returns the configuration-defined password.
func ConfigPassword() string {
	if ConfigurationAuthInfoChanged() {
		return config.AuthInfo.Password
	}

	return ""
}

// ApiTokenFilePath returns the configuration-defined API Token file path.
func ConfigApiTokenFilePath() string {
	if ConfigurationAuthInfoChanged() {
		return config.AuthInfo.ApiTokenFilePath
	}

	return ""
}

// InlineApiToken returns the configuration-defined ApiToken.
func ConfigInlineApiToken() string {
	if ConfigurationAuthInfoChanged() {
		return config.AuthInfo.ApiToken
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
func readConfig(path string) (*InstanceConfig, error) {
	var conf InstanceConfig

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

	fmt.Println("Here")

	fmt.Printf("this is the configuration file struct: %+v\n", conf)
	fmt.Printf("this is the instances struct: %+v\n", *conf.Instances)
	fmt.Printf("this is the authinfo struct: %+v\n", *conf.AuthInfo)
	fmt.Printf("this is the contexts struct: %+v\n", *conf.Contexts)
	fmt.Printf("this is the current context: %+v\n", conf.CurrContext)

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

	config = instance

	// TODO(2): Edit for when user submits multiple Instances
	if config.Instances != nil {
		configInstancesSet = pointers.Bool(true)
		config.Instances.digest()
	}

	// TODO(2): Edit for when user submits multiple AuthInfo
	if config.AuthInfo != nil {
		configAuthInfoSet = pointers.Bool(true)
		config.AuthInfo.digest()
	}

	// TODO(2): Edit for when user submits multiple Contexts
	if config.Contexts != nil {
		configContextsSet = pointers.Bool(true)
		config.Contexts.digest()
	}

	if config.CurrContext != "" {
		configCurrContextSet = pointers.Bool(true)
	}

	return nil
}

// Instance.digest is a utility function that, given a particular instance, adjusts various flags
func (i Instance) digest() {
	if i.Endpoint != "" {
		configEndpointSet = pointers.Bool(true)
	}

	if i.CaCertificatePath != "" {
		configCaCertificatePathSet = pointers.Bool(true)
	}
	if i.CaCertificate != "" {
		configInlineCaCertificateSet = pointers.Bool(true)
	}
	if i.Plaintext != "" {
		configPlaintextSet = pointers.Bool(true)
	}
	if i.DirectGRPC {
		configDirectGRPCSet = pointers.Bool(true)
	}
	if i.ForceHTTP1 {
		configForceHTTP1Set = pointers.Bool(true)
	}
	if i.Insecure {
		configUseInsecureSet = pointers.Bool(true)
	}
	if i.InsecureSkipTLSVerify {
		configInsecureSkipTLSVerifySet = pointers.Bool(true)
	}
}

// AuthInfo.digest is a utility function that, given particular AuthInfo, adjusts various flags
func (a AuthInfo) digest() {
	if a.Username != "" {
		configUsernameSet = pointers.Bool(true)
	}

	if a.Password != "" {
		configPasswordSet = pointers.Bool(true)
	}

	if a.ApiToken != "" {
		configInlineApiTokenSet = pointers.Bool(true)
	}

	if a.ApiTokenFilePath != "" {
		configApiTokenFilePathSet = pointers.Bool(true)
	}
}

// Contexts.digest is a utility function that, given a particular Context, adjusts flags
func (c Context) digest() {
	if c.Instance != "" {
		configContextInstanceSet = pointers.Bool(true)
	}

	if c.AuthInfo != "" {
		configContextsAuthInfoSet = pointers.Bool(true)
	}
}
