package client

import (
	"github.com/stackrox/rox/pkg/mtls"
)

var defaultOptions = options{
	mtlsSubject:   mtls.ScannerSubject,
	address:       ":8443",
	serverName:    "scanner-v4.stackrox",
	withTLSVerify: true,
}

// Option configures the options to create a scanner client.
type Option func(*options)

type options struct {
	mtlsSubject   mtls.Subject
	address       string
	serverName    string
	withTLSVerify bool
}

// GetCA loads and instantiates a Stackrox CA from the options specified.
func (o *options) GetCA() (mtls.CA, error) {
	// Options only support the default CA for now.
	return mtls.LoadDefaultCA()
}

// WithSubject specifies the mTLS subject to use.
func WithSubject(subject mtls.Subject) Option {
	return func(o *options) {
		o.mtlsSubject = subject
	}
}

// WithServerName specifies the mTLS server name used to verify the server's certificate.
func WithServerName(serverName string) Option {
	return func(o *options) {
		o.serverName = serverName
	}
}

// WithoutTLSVerify disables TLS verification, and don't read or use client
// certificates (mTLS).
func WithoutTLSVerify(o *options) {
	o.withTLSVerify = false
}

// WithAddress specifies the GRPC address to connect.
func WithAddress(address string) Option {
	return func(o *options) {
		o.address = address
	}
}

func makeOptions(opts ...Option) options {
	o := defaultOptions
	for _, opt := range opts {
		opt(&o)
	}
	return o
}
