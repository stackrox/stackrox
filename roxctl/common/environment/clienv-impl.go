package environment

import (
	"github.com/stackrox/rox/roxctl/common"
	"google.golang.org/grpc"
)

type cliEnvironmentImpl struct {
	io     IO
	client common.RoxctlHTTPClient
}

// NewCLIEnvironment creates a new CLI environment with the given IO and common.RoxctlHTTPClient
func NewCLIEnvironment(io IO, client common.RoxctlHTTPClient) *cliEnvironmentImpl {
	return &cliEnvironmentImpl{io: io, client: client}
}

// HTTPClient returns the common.RoxctlHTTPClient associated with the CLI Environment
func (c *cliEnvironmentImpl) HTTPClient() common.RoxctlHTTPClient {
	return c.client
}

// GRPCConnection returns the common.GetGRPCConnection
func (c *cliEnvironmentImpl) GRPCConnection() (*grpc.ClientConn, error) {
	return common.GetGRPCConnection()
}

// InputOutput returns the IO associated with the CLI Environment which holds all relevant input / output streams
func (c *cliEnvironmentImpl) InputOutput() IO {
	return c.io
}
