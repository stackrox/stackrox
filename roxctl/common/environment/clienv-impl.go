package environment

import (
	"io"
	"time"

	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/printer"
	"google.golang.org/grpc"
)

type cliEnvironmentImpl struct {
	io              IO
	logger          Logger
	colorfulPrinter printer.ColorfulPrinter
}

// NewCLIEnvironment creates a new CLI environment with the given IO and common.RoxctlHTTPClient
func NewCLIEnvironment(io IO, c printer.ColorfulPrinter) Environment {
	return &cliEnvironmentImpl{
		io:              io,
		colorfulPrinter: c,
		logger:          NewLogger(io, c),
	}
}

// HTTPClient returns the common.RoxctlHTTPClient associated with the CLI Environment
func (c *cliEnvironmentImpl) HTTPClient(timeout time.Duration) (common.RoxctlHTTPClient, error) {
	return common.GetRoxctlHTTPClient(timeout, flags.ForceHTTP1(), flags.UseInsecure())
}

// GRPCConnection returns the common.GetGRPCConnection
func (c *cliEnvironmentImpl) GRPCConnection() (*grpc.ClientConn, error) {
	return common.GetGRPCConnection()
}

// InputOutput returns the IO associated with the CLI Environment which holds all relevant input / output streams
func (c *cliEnvironmentImpl) InputOutput() IO {
	return c.io
}

func (c *cliEnvironmentImpl) Logger() Logger {
	return c.logger
}

func (c *cliEnvironmentImpl) ColorWriter() io.Writer {
	return colorWriter{
		colorfulPrinter: c.colorfulPrinter,
		out:             c.InputOutput().Out,
	}
}

// ConnectNames returns the endpoint and (SNI) server name
func (c *cliEnvironmentImpl) ConnectNames() (string, string, error) {
	return common.ConnectNames()
}

type colorWriter struct {
	colorfulPrinter printer.ColorfulPrinter
	out             io.Writer
}

func (w colorWriter) Write(p []byte) (int, error) {
	n, err := w.out.Write([]byte(w.colorfulPrinter.ColorWords(string(p))))
	if err != nil {
		return n, err
	}
	return len(p), nil
}
