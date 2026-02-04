package environment

import (
	"io"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/config"
	"github.com/stackrox/rox/roxctl/common/flags"
	cliIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	authOptionsListMessage = `  - Use the --password flag or set the ROX_ADMIN_PASSWORD environment variable
  - Use the --token-file flag and point to a file containing your API token
  - Set the ROX_API_TOKEN environment variable with your API token
  - Run "roxctl central login" to save credentials (requires writable home directory)`

	missingAuthCredsMessage = `No authentication credentials are available. Please provide authentication using one of the following methods:
` + authOptionsListMessage
)

type cliEnvironmentImpl struct {
	io              cliIO.IO
	logger          logger.Logger
	colorfulPrinter printer.ColorfulPrinter
}

var (
	singleton Environment
	once      sync.Once

	errInvalidCombination = errox.InvalidArgs.New("cannot use basic and token-based authentication at the same time")
)

// NewTestCLIEnvironment creates a new CLI environment with the given IO and common.RoxctlHTTPClient.
// It should be only used within tests.
func NewTestCLIEnvironment(_ *testing.T, io cliIO.IO, c printer.ColorfulPrinter) Environment {
	return &cliEnvironmentImpl{
		io:              io,
		colorfulPrinter: c,
		logger:          logger.NewLogger(io, c),
	}
}

// CLIEnvironment creates a new default CLI environment.
func CLIEnvironment() Environment {
	// We have chicken and egg problem here. We need to parse flags to know if --no-color was set
	// but at the same time we need to set printer to handle possible flags parsing errors.
	// Instead of using native cobra flags mechanism we can just check if os.Args contains --no-color.
	once.Do(func() {
		var colorPrinter printer.ColorfulPrinter
		if flags.HasNoColor(os.Args) {
			colorPrinter = printer.NoColorPrinter()
		} else {
			colorPrinter = printer.DefaultColorPrinter()
		}
		environIO := cliIO.DefaultIO()
		singleton = &cliEnvironmentImpl{
			io:              environIO,
			colorfulPrinter: colorPrinter,
			logger:          logger.NewLogger(environIO, colorPrinter),
		}
	})
	return singleton
}

// HTTPClient returns the common.RoxctlHTTPClient associated with the CLI Environment
func (c *cliEnvironmentImpl) HTTPClient(timeout time.Duration, options ...common.HttpClientOption) (common.RoxctlHTTPClient, error) {
	config := common.NewHttpClientConfig(
		common.WithTimeout(timeout),
		common.WithLogger(c.Logger()),
	)

	for _, optFunc := range options {
		optFunc(config)
	}

	if config.AuthMethod == nil {
		var err error
		config.AuthMethod, err = determineAuthMethod(c)
		if err != nil {
			return nil, errors.Wrap(err, "determining auth method")
		}
	}
	client, err := common.GetRoxctlHTTPClient(config)
	return client, errors.WithStack(err)
}

// GRPCConnection returns the common.GetGRPCConnection
func (c *cliEnvironmentImpl) GRPCConnection(connectionOpts ...common.GRPCOption) (*grpc.ClientConn, error) {
	am, err := determineAuthMethod(c)
	if err != nil {
		return nil, errors.Wrap(err, "determining auth method")
	}
	connection, err := common.GetGRPCConnection(am, connectionOpts...)
	return connection, errors.WithStack(err)
}

// InputOutput returns the IO associated with the CLI Environment which holds all relevant input / output streams
func (c *cliEnvironmentImpl) InputOutput() cliIO.IO {
	return c.io
}

func (c *cliEnvironmentImpl) Logger() logger.Logger {
	return c.logger
}

func (c *cliEnvironmentImpl) ColorWriter() io.Writer {
	return colorWriter{
		colorfulPrinter: c.colorfulPrinter,
		out:             c.InputOutput().Out(),
	}
}

// ConnectNames returns the endpoint and (SNI) server name
func (c *cliEnvironmentImpl) ConnectNames() (string, string, error) {
	names, s, _, err := common.ConnectNames()
	return names, s, errors.Wrap(err, "could not get endpoint")
}

// ConfigStore returns a config.Store capable of reading / writing configuration for roxctl.
func (c *cliEnvironmentImpl) ConfigStore() (config.Store, error) {
	cfgStore, err := config.NewConfigStore()
	if err != nil {
		return nil, errors.Wrap(err, "creating config store")
	}
	return cfgStore, nil
}

type colorWriter struct {
	colorfulPrinter printer.ColorfulPrinter
	out             io.Writer
}

func (w colorWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write([]byte(w.colorfulPrinter.ColorWords(string(p))))
	if err != nil {
		return n, errors.Wrap(err, "could not write")
	}
	return len(p), nil
}

func determineAuthMethod(cliEnv Environment) (auth.Method, error) {
	// First check if explicit auth was provided (password, token-file, or ROX_API_TOKEN)
	explicitMethod, err := determineAuthMethodExt(
		flags.APITokenFileChanged(), flags.PasswordChanged(),
		flags.APITokenFile() == "", flags.Password() == "", env.TokenEnv.Setting() == "")

	if err != nil {
		return nil, err
	}

	if explicitMethod != nil {
		return explicitMethod, nil
	}

	// No explicit auth was provided - fall back to saved credentials from "roxctl central login"
	// Wrap ConfigMethod to provide helpful guidance if it fails
	return &configMethodWithGuidance{
		wrapped: ConfigMethod(cliEnv),
	}, nil
}

// configMethodWithGuidance wraps ConfigMethod to provide helpful error messages
// when no explicit authentication was provided and saved credentials fail to load
type configMethodWithGuidance struct {
	wrapped auth.Method
}

func (c *configMethodWithGuidance) Type() string {
	return c.wrapped.Type()
}

func (c *configMethodWithGuidance) GetCredentials(url string) (credentials.PerRPCCredentials, error) {
	creds, err := c.wrapped.GetCredentials(url)
	if err == nil {
		return creds, nil
	}

	// Check if this is already a NoCredentials error (from auth_config.go when no saved credentials exist)
	if errors.Is(err, errox.NoCredentials) {
		// Replace with the full list of auth options since no explicit auth was provided
		// (avoiding duplication of "roxctl central login" message from the original error)
		return nil, errox.NoCredentials.New(missingAuthCredsMessage)
	}

	// Some other error occurred (filesystem permissions, config parsing, etc.)
	// Provide context that no explicit auth was provided and suggest alternatives
	return nil, errors.Wrapf(err,
		"no explicit authentication credentials were provided, and failed to retrieve saved credentials from roxctl config."+
			" Provide authentication using one of the following methods:\n%s",
		authOptionsListMessage)
}

func determineAuthMethodExt(tokenFileChanged, passwordChanged, tokenFileNameEmpty, passwordEmpty, tokenEmpty bool) (auth.Method, error) {
	// Prefer command line arguments over environment variables.
	switch {
	case tokenFileChanged && tokenFileNameEmpty || passwordChanged && passwordEmpty:
		utils.Should(errox.InvariantViolation)
		return nil, nil
	case tokenFileChanged && passwordChanged:
		return nil, errInvalidCombination
	case !(tokenFileChanged || passwordChanged || tokenFileNameEmpty || passwordEmpty):
		return nil, errInvalidCombination
	case !(tokenFileChanged || passwordChanged || tokenEmpty || passwordEmpty):
		return nil, errInvalidCombination
	case passwordChanged || !(passwordEmpty || tokenFileChanged):
		return auth.BasicAuth(), nil
	case tokenFileChanged || !tokenFileNameEmpty || !tokenEmpty:
		return auth.TokenAuth(), nil
	default:
		return nil, nil
	}
}
