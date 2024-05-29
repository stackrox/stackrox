package client

import (
	"fmt"
	"net"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/mtls"
)

var (
	defaultOptions = options{
		indexerOpts: connOptions{
			mTLSSubject:   mtls.ScannerV4IndexerSubject,
			address:       ":8443",
			serverName:    fmt.Sprintf("scanner-v4-indexer.%s.svc", env.Namespace.Setting()),
			skipTLSVerify: false,
		},
		matcherOpts: connOptions{
			mTLSSubject:   mtls.ScannerV4MatcherSubject,
			address:       ":8443",
			serverName:    fmt.Sprintf("scanner-v4-matcher.%s.svc", env.Namespace.Setting()),
			skipTLSVerify: false,
		},
		comboMode: false,
	}
)

// Option configures the options to create a scanner client.
type Option func(*options)

type connOptions struct {
	mTLSSubject   mtls.Subject
	address       string
	serverName    string
	skipTLSVerify bool
}

type options struct {
	indexerOpts connOptions
	matcherOpts connOptions
	comboMode   bool
}

// ImageRegistryOpt defines options for reaching out to image registries.
type ImageRegistryOpt struct {
	InsecureSkipTLSVerify bool
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
	o.indexerOpts.skipTLSVerify = true
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
	o.matcherOpts.skipTLSVerify = true
}

// WithMatcherAddress specifies the gRPC address to connect.
func WithMatcherAddress(address string) Option {
	return func(o *options) {
		o.matcherOpts.address = address
	}
}

func makeOptions(opts ...Option) (options, error) {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	// If both indexer and matcher are equal, we are in combo mode. Right now structs
	// are simple enough to compare.
	o.comboMode = o.indexerOpts == o.matcherOpts
	return o, validateOptions(o)
}

func validateOptions(o options) error {
	// If this check is removed, make sure we still properly use the DNS name resolver.
	if _, _, err := net.SplitHostPort(o.indexerOpts.address); err != nil {
		return fmt.Errorf("invalid indexer address (want [host]:port): %w", err)
	}
	// If this check is removed, make sure we still properly use the DNS name resolver.
	if _, _, err := net.SplitHostPort(o.matcherOpts.address); err != nil {
		return fmt.Errorf("invalid matcher address (want [host]:port): %w", err)
	}
	return nil
}
