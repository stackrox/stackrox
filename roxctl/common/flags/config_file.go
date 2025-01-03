package flags

import (
	"fmt"
	"os"
	"os/user"
	"syscall"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/pointers"
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

// DefaultConfig is a utility that returns the default configuration template.
func DefaultConfig() *InstanceConfig {
	return &InstanceConfig{
		Instances: &Instance{
			Endpoint:              "",
			CaCertificatePath:     "",
			CaCertificate:         "",
			Plaintext:             false,
			DirectGRPC:            false,
			ForceHTTP1:            false,
			Insecure:              false,
			InsecureSkipTLSVerify: false,
		},
	}
}

var (
	configFile    string
	configFileSet = pointers.Bool(false)
	config        = DefaultConfig()

	// Flags related to the Instance struct
	configInstancesSet             = pointers.Bool(false) // Existence flag
	configEndpointSet              = pointers.Bool(false)
	configCaCertificatePathSet     = pointers.Bool(false)
	configInlineCaCertificateSet   = pointers.Bool(false)
	configPlaintextSet             = pointers.Bool(false)
	configDirectGRPCSet            = pointers.Bool(false)
	configForceHTTP1Set            = pointers.Bool(false)
	configUseInsecureSet           = pointers.Bool(false)
	configInsecureSkipTLSVerifySet = pointers.Bool(false)

	// Flags related to the AuthInfo struct
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

// // ConfigurationFileChanged returns whether the configuration file is provided as an argument.
func ConfigurationFileChanged() bool {
	return configFileSet != nil && *configFileSet
}

// Endpoint returns the configuration-defined endpoint.
func ConfigEndpoint() string {
	return config.Instances.Endpoint
}

// CaCertificatePath returns the configuration-defined CA Certificate path.
func ConfigCaCertificatePath() string {
	return config.Instances.CaCertificatePath
}

// CaCertificate returns the configuration-defined inline CA Certificate.
func ConfigInlineCaCertificate() string {
	return config.Instances.CaCertificate
}

// Plaintext returns the configuration-defined plaintext.
func ConfigPlaintext() bool {
	return config.Instances.Plaintext
}

// DirectGRPC returns the configuration-defined Direct GRPC option.
func ConfigUseDirectGRPC() bool {
	return config.Instances.DirectGRPC
}

// ForceHTTP1 returns the configuration-defined Force HTTP option.
func ConfigForceHTTP1() bool {
	return config.Instances.ForceHTTP1
}

// ConfigUseInsecure returns the configuration-defined Insecure option.
func ConfigUseInsecure() bool {
	return config.Instances.Insecure
}

// InsecureSkipTLSVerify returns the configuration-defined Insecure Skip TLS Verify option.
func ConfigSkipTLSValidation() bool {
	return config.Instances.InsecureSkipTLSVerify
}

// Username returns the configuration-defined username.
func ConfigUsername() string {
	return config.AuthInfo.Username
}

// Password returns the configuration-defined password.
func ConfigPassword() string {
	return config.AuthInfo.Password
}

// ApiTokenFilePath returns the configuration-defined API Token file path.
func ConfigApiTokenFilePath() string {
	return config.AuthInfo.ApiTokenFilePath
}

// InlineApiToken returns the configuration-defined ApiToken.
func ConfigInlineApiToken() string {
	return config.AuthInfo.ApiToken
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

// loadEnvSettings is a utility for populating the configuration struct with environment settings.
func loadEnvSettings() {
	clientForceHTTP1Env := env.ClientForceHTTP1Env.BooleanSetting()
	directGRPCEnv := env.DirectGRPCEnv.BooleanSetting()
	insecureClientEnv := env.InsecureClientEnv.BooleanSetting()
	insecureClientSkipTLSVerifyEnv := env.InsecureClientSkipTLSVerifyEnv.BooleanSetting()
	plaintextEnv := env.PlaintextEnv.BooleanSetting()

	if clientForceHTTP1Env {
		config.Instances.ForceHTTP1 = clientForceHTTP1Env
	}

	if directGRPCEnv {
		config.Instances.DirectGRPC = directGRPCEnv
	}

	if insecureClientEnv {
		config.Instances.Insecure = insecureClientEnv
	}

	if insecureClientSkipTLSVerifyEnv {
		config.Instances.InsecureSkipTLSVerify = insecureClientSkipTLSVerifyEnv
	}

	if plaintextEnv {
		config.Instances.Plaintext = plaintextEnv
	}
}

// loadFlags is a utility for populating the configuration struct with environment settings.
func loadFlags() {
	clientForceHTTP1Env := ForceHTTP1()
	directGRPCEnv := UseDirectGRPC()
	insecureClientEnv := UseInsecure()
	insecureClientSkipTLSVerifyEnv := SkipTLSValidation()

	if insecureClientSkipTLSVerifyEnv == nil {
		insecureClientSkipTLSVerifyEnv = pointers.Bool(false)
	}
	plaintextEnv := booleanFlagOrSettingValue(plaintext, *plaintextSet, env.PlaintextEnv)

	if clientForceHTTP1Env {
		config.Instances.ForceHTTP1 = clientForceHTTP1Env
	}

	if directGRPCEnv {
		config.Instances.DirectGRPC = directGRPCEnv
	}

	if insecureClientEnv {
		config.Instances.Insecure = insecureClientEnv
	}

	if *insecureClientSkipTLSVerifyEnv {
		config.Instances.InsecureSkipTLSVerify = *insecureClientSkipTLSVerifyEnv
	}

	if plaintextEnv {
		config.Instances.Plaintext = plaintextEnv
	}
}

// readConfig is a utilty for reading YAML-based configuration files
func readConfig(path string) error {

	err := checkFilePermissionsAndOwnership(path)

	if err != nil {
		return errors.Wrapf(err, "file permission or ownership error: %v", err)
	}

	bytes, err := os.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to retrieve config from file %q", path)
	}
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		return errors.Wrapf(err, "failed to retrieve config from file %q", path)
	}

	fmt.Println("Here")

	fmt.Printf("this is the configuration file struct: %+v\n", config)
	fmt.Printf("this is the instances struct: %+v\n", *config.Instances)
	fmt.Printf("this is the authinfo struct: %+v\n", *config.AuthInfo)
	fmt.Printf("this is the contexts struct: %+v\n", *config.Contexts)
	fmt.Printf("this is the current context: %+v\n", config.CurrContext)

	return nil
}

// Load loads a config file from a given path
//   - Load will prioritize the values that are defined within
//     the configuration files over variables defined within the environment
func LoadConfig(cmd *cobra.Command, args []string) error {

	if configFile == "" || !ConfigurationFileChanged() {
		return nil
	}

	err := readConfig(configFile)

	if err != nil {
		log.Errorf("Error reading instance config file: %v", err)
		return err
	}

	// populate config with non-default environment settings
	loadEnvSettings()

	// populate config with non-default flags
	loadFlags()

	// TODO(2): Edit for when user submits multiple Instances
	if config.Instances != nil {
		configInstancesSet = pointers.Bool(true)
		config.Instances.digest()
	}

	// // TODO(2): Edit for when user submits multiple AuthInfo
	// if config.AuthInfo != nil {
	// 	configAuthInfoSet = pointers.Bool(true)
	// 	config.AuthInfo.digest()
	// }

	// // TODO(2): Edit for when user submits multiple Contexts
	// if config.Contexts != nil {
	// 	configContextsSet = pointers.Bool(true)
	// 	config.Contexts.digest()
	// }

	// if config.CurrContext != "" {
	// 	configCurrContextSet = pointers.Bool(true)
	// }

	return nil
}

// Instance.digest is a utility function that, given a particular instance, adjusts various flags
func (i Instance) digest() {
	if i.Endpoint != "" {
		configEndpointSet = pointers.Bool(true)
	}

	if config.CurrContext != "" {
		configCurrContextSet = pointers.Bool(true)
	}

	if i.CaCertificatePath != "" {
		configCaCertificatePathSet = pointers.Bool(true)
	}
	if i.CaCertificate != "" {
		configInlineCaCertificateSet = pointers.Bool(true)
	}
	if i.Plaintext {
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
