package environment

import (
	"io"
	"time"

	"github.com/stackrox/rox/roxctl/common"
	commonIO "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"google.golang.org/grpc"
)

// Environment abstracts the interfaces.RoxctlHTTPClient, IO and grpc.ClientConn used within each command of the CLI.
//go:generate mockgen-wrapper
type Environment interface {
	// HTTPClient returns a interfaces.RoxctlHTTPClient
	HTTPClient(timeout time.Duration) (common.RoxctlHTTPClient, error)

	// GRPCConnection returns an authenticated grpc.ClientConn
	GRPCConnection() (*grpc.ClientConn, error)

	// InputOutput returns an IO which holds all input / output streams
	InputOutput() commonIO.IO

	// Logger returns Logger which handle all output
	Logger() logger.Logger

	// ColorWriter returns io.Writer that colorize bytes and writes them to InputOutput().Out
	ColorWriter() io.Writer

	// ConnectNames returns the endpoint and (SNI) server name
	ConnectNames() (string, string, error)
}
