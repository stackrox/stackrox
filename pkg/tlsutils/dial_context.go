package tlsutils

import (
	"context"
	"crypto/tls"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DialContext attempts to establishes a TLS-enabled connection in a context-aware manner.
func DialContext(ctx context.Context, network, addr string, tlsConfig *tls.Config) (*tls.Conn, error) {
	dialer := tls.Dialer{
		Config: tlsConfig,
	}
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		log.Debugw("tls dial failed", logging.Err(err))

		return nil, errors.Wrap(errox.ConcealSensitive(err), "unable to establish a TLS-enabled connection")
	}
	return conn.(*tls.Conn), nil
}
