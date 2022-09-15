package environment

import (
	"io"
	"net/url"
	"time"

	"github.com/stackrox/rox/roxctl/common"
	"github.com/stackrox/rox/roxctl/common/auth"
	io2 "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"google.golang.org/grpc"
)

// Environment abstracts the interfaces.RoxctlHTTPClient, IO and grpc.ClientConn used within each command of the CLI.
//
//go:generate mockgen-wrapper
type Environment interface {
	// HTTPClient returns a interfaces.RoxctlHTTPClient
	HTTPClient(timeout time.Duration, authOpt ...auth.Method) (common.RoxctlHTTPClient, error)

	// GRPCConnection returns an authenticated grpc.ClientConn
	GRPCConnection(authOpt ...auth.Method) (*grpc.ClientConn, error)

	// InputOutput returns an IO which holds all input / output streams
	InputOutput() io2.IO

	// Logger returns Logger which handle all output
	Logger() logger.Logger

	// ColorWriter returns io.Writer that colorize bytes and writes them to InputOutput().Out
	ColorWriter() io.Writer

	// ConnectNames returns the endpoint and (SNI) server name
	ConnectNames() (string, string, error)

	BaseURL() *url.URL
}
