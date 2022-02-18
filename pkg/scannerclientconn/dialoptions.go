package scannerclientconn

import "crypto/tls"

// DialOptions specifies how to configure the connection with Scanner.
type DialOptions struct {
	// TLSConfig specifies the TLS configuration to use to talk to Scanner.
	// If nil, then an insecure connection is used.
	TLSConfig *tls.Config
}
