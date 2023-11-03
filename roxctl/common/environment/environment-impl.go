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
	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/config"
	"github.com/stackrox/rox/roxctl/common/flags"
	cliIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"google.golang.org/grpc"
)

type cliEnvironmentImpl struct {
	io              cliIO.IO
	logger          logger.Logger
	colorfulPrinter printer.ColorfulPrinter
}

var (
	singleton Environment
	once      sync.Once
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
		singleton = &cliEnvironmentImpl{
			io:              cliIO.DefaultIO(),
			colorfulPrinter: colorPrinter,
			logger:          logger.NewLogger(cliIO.DefaultIO(), colorPrinter),
		}
	})
	return singleton
}

// HTTPClient returns the common.RoxctlHTTPClient associated with the CLI Environment
func (c *cliEnvironmentImpl) HTTPClient(timeout time.Duration, authMethod ...auth.Method) (common.RoxctlHTTPClient, error) {
	var am auth.Method
	if len(authMethod) > 0 {
		am = authMethod[0]
	} else {
		var err error
		am, err = determineAuthMethod(c)
		if err != nil {
			return nil, errors.Wrap(err, "determining auth method")
		}
	}
	client, err := common.GetRoxctlHTTPClient(am, timeout, flags.ForceHTTP1(), flags.UseInsecure(), c.Logger())
	return client, errors.WithStack(err)
}

// GRPCConnection returns the common.GetGRPCConnection
func (c *cliEnvironmentImpl) GRPCConnection(connectionOpts ...common.GRPCOption) (*grpc.ClientConn, error) {
	am, err := determineAuthMethod(c)
	if err != nil {
		return nil, errors.Wrap(err, "determining auth method")
	}
	connection, err := common.GetGRPCConnection(am, c.Logger(), connectionOpts...)
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
	if flags.APITokenFile() != "" && flags.Password() != "" {
		return nil, errox.InvalidArgs.New("cannot use basic and token-based authentication at the same time")
	}
	switch {
	case flags.Password() != "":
		return auth.BasicAuth(), nil
	case flags.APITokenFile() != "" || env.TokenEnv.Setting() != "":
		return auth.TokenAuth(), nil
	default:
		return ConfigMethod(cliEnv), nil
	}
}
