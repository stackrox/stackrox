package tlsutils

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

func DialContextWithRetries(ctx context.Context, network, addr string, tlsConfig *tls.Config) (*tls.Conn, error) {
	dialer := tls.Dialer{
		Config: tlsConfig,
	}
	return DialContextWithRetriesWithDialer(ctx, dialer, network, addr)
}

func isPermanentError(err error) bool {
	return errors.As(err, &x509.CertificateInvalidError{}) ||
		errors.As(err, &x509.HostnameError{}) ||
		errors.As(err, &x509.UnknownAuthorityError{}) ||
		errors.As(err, &x509.SystemRootsError{})
}

func DialContextWithRetriesWithDialer(ctx context.Context, dialer tls.Dialer, network, addr string) (*tls.Conn, error) {
	expBackoff := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(2*time.Second),
		backoff.WithMultiplier(2),
		backoff.WithMaxElapsedTime(0), // We will simply use the deadline of the provided context instead.
	)
	expBackoffWithCtx := backoff.WithContext(expBackoff, ctx)
	var dialConn net.Conn
	var dialErr error

	err := backoff.RetryNotify(func() error {
		dialConn, dialErr = DialContextWithDialer(ctx, dialer, network, addr)
		if dialErr != nil {
			if isPermanentError(dialErr) {
				log.Warnf("tls dial failed due to permanent error %v, not retrying", dialErr)
				return backoff.Permanent(dialErr)
			}
			return dialErr
		}
		return nil
	}, expBackoffWithCtx, func(err error, d time.Duration) {
		log.Warnf("tls dial failed: %v, retrying after %s", err, d.Round(time.Second))
	})
	if err != nil {
		return nil, multierror.Append(err, dialErr)
	}

	return dialConn.(*tls.Conn), nil
}

// DialContext attempts to establishes a TLS-enabled connection in a context-aware manner.
func DialContext(ctx context.Context, network, addr string, tlsConfig *tls.Config) (*tls.Conn, error) {
	dialer := tls.Dialer{
		Config: tlsConfig,
	}
	return DialContextWithDialer(ctx, dialer, network, addr)
}

// DialContextWithDialer attempts to establishes a TLS-enabled connection in a context-aware manner using
// provided tls Dialer.
func DialContextWithDialer(ctx context.Context, dialer tls.Dialer, network, addr string) (*tls.Conn, error) {
	conn, err := dialer.DialContext(ctx, network, addr)
	if err != nil {
		log.Debugw("tls dial failed", logging.Err(err))

		return nil, errors.Wrap(errox.ConcealSensitive(err), "unable to establish a TLS-enabled connection")
	}
	return conn.(*tls.Conn), nil
}
