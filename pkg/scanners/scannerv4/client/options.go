package client

import (
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	defaultConnOptions = connOptions{
		mTLSSubject: mtls.ScannerSubject,
		address:     ":8443",
		serverName:  "scanner-v4.stackrox",
		skipTLS:     false,
	}

	defaultOptions = options{
		indexerOpts: defaultConnOptions,
		matcherOpts: defaultConnOptions,
		comboMode:   true,
	}
)

// Option configures the options to create a scanner client.
type Option func(*options)

type connOptions struct {
	mTLSSubject mtls.Subject
	address     string
	serverName  string
	skipTLS     bool
}

type options struct {
	indexerOpts connOptions
	matcherOpts connOptions
	comboMode   bool
}

// WithSubject specifies the mTLS subject to use.
func WithSubject(subject mtls.Subject) Option {
	return func(o *options) {
		WithIndexerSubject(subject)(o)
		WithMatcherSubject(subject)(o)
	}
}

// WithServerName specifies the mTLS server name used to verify the server's certificate.
func WithServerName(serverName string) Option {
	return func(o *options) {
		WithIndexerServerName(serverName)(o)
		WithMatcherServerName(serverName)(o)
	}
}

// SkipTLSVerification disables TLS verification, preventing the reading and usage
// of client certificates (mTLS).
func SkipTLSVerification(o *options) {
	SkipIndexerTLSVerification(o)
	SkipMatcherTLSVerification(o)
}

// WithAddress specifies the gRPC address to connect.
func WithAddress(address string) Option {
	return func(o *options) {
		WithIndexerAddress(address)(o)
		WithMatcherAddress(address)(o)
	}
}

// WithIndexerSubject specifies the mTLS subject to use.
func WithIndexerSubject(subject mtls.Subject) Option {
	return func(o *options) {
		o.indexerOpts.mTLSSubject = subject
	}
}

// WithIndexerServerName specifies the mTLS server name used to verify the server's certificate.
func WithIndexerServerName(serverName string) Option {
	return func(o *options) {
		o.indexerOpts.serverName = serverName
	}
}

// SkipIndexerTLSVerification disables TLS verification, preventing the reading and usage
// of client certificates (mTLS).
func SkipIndexerTLSVerification(o *options) {
	o.indexerOpts.skipTLS = true
}

// WithIndexerAddress specifies the gRPC address to connect.
func WithIndexerAddress(address string) Option {
	return func(o *options) {
		o.indexerOpts.address = address
	}
}

// WithMatcherSubject specifies the mTLS subject to use.
func WithMatcherSubject(subject mtls.Subject) Option {
	return func(o *options) {
		o.matcherOpts.mTLSSubject = subject
	}
}

// WithMatcherServerName specifies the mTLS server name used to verify the server's certificate.
func WithMatcherServerName(serverName string) Option {
	return func(o *options) {
		o.matcherOpts.serverName = serverName
	}
}

// SkipMatcherTLSVerification disables TLS verification, preventing the reading and usage
// of client certificates (mTLS).
func SkipMatcherTLSVerification(o *options) {
	o.matcherOpts.skipTLS = true
}

// WithMatcherAddress specifies the gRPC address to connect.
func WithMatcherAddress(address string) Option {
	return func(o *options) {
		o.matcherOpts.address = address
	}
}

func makeOptions(opts ...Option) options {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	// If both indexer and matcher are equal, we are in combo mode. Right now structs
	// are simple enough to compare.
	o.comboMode = o.indexerOpts == o.matcherOpts
	return o
}
