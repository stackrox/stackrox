package tlsutils

import (
	"context"
	"crypto/tls"
)

// DialContext attempts to establishes a TLS-enabled connection in a context-aware manner.
func DialContext(ctx context.Context, network, addr string, tlsConfig *tls.Config) (*tls.Conn, error) {
	dialer := tls.Dialer{
		Config: tlsConfig,
	}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}
	return conn.(*tls.Conn), nil
}
