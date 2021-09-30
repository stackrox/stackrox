package environment

import (
	"time"

	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/flags"
	"google.golang.org/grpc"
)

type cliEnvironmentImpl struct {
	io IO
}

// NewCLIEnvironment creates a new CLI environment with the given IO and common.RoxctlHTTPClient
func NewCLIEnvironment(io IO) *cliEnvironmentImpl {
	return &cliEnvironmentImpl{io: io}
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
