package common

import (
	"io"
	"time"

	"google.golang.org/grpc"
)

// Environment abstracts the interfaces.RoxctlHTTPClient, IO and grpc.ClientConn used within each command of the CLI.
//go:generate mockgen-wrapper
type Environment interface {
	// HTTPClient returns a interfaces.RoxctlHTTPClient
	HTTPClient(timeout time.Duration) (RoxctlHTTPClient, error)

	// GRPCConnection returns an authenticated grpc.ClientConn
	GRPCConnection() (*grpc.ClientConn, error)

	// InputOutput returns an IO which holds all input / output streams
	InputOutput() IO

	// Logger returns Logger which handle all output
	Logger() Logger

	// ColorWriter returns io.Writer that colorize bytes and writes them to InputOutput().Out
	ColorWriter() io.Writer

	// ConnectNames returns the endpoint and (SNI) server name
	ConnectNames() (string, string, error)
}

// IO holds information about io streams used within commands of roxctl.
type IO interface {
	In() io.Reader
	Out() io.Writer
	ErrOut() io.Writer
}

// Logger is a struct responsible for printing messages. It should be preferred over fmt functions.
type Logger interface {
	// ErrfLn prints a formatted string with a newline, prefixed with ERROR and colorized
	ErrfLn(format string, a ...interface{})

	// WarnfLn prints a formatted string with a newline, prefixed with WARN and colorized
	WarnfLn(format string, a ...interface{})

	// InfofLn prints a formatted string with a newline, prefixed with INFO and colorized
	InfofLn(format string, a ...interface{})

	// PrintfLn prints a formatted string with newline at the end
	PrintfLn(format string, a ...interface{})
}
