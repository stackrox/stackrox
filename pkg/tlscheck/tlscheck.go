package tlscheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	timeout = 2 * time.Second
)

// validateWithScheme ensures that the URL: (1) has scheme, (2) passes the validation of url.Parse.
// If validation passes, the returned result contains scheme `tcp://` prepended to the input string.
func validateWithScheme(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	// We rerun this prependig the `tcp://` scheme, due to the following comment on url.Parse:
	// "Trying to parse a hostname and path without a scheme
	// is invalid but may not necessarily return an error, due to parsing ambiguities."
	if !u.IsAbs() || u.Host == "" {
		return validateWithScheme("tcp://" + s)
	}
	return u.String(), nil
}

// CheckTLS checks if the address is using TLS
func CheckTLS(ctx context.Context, origAddr string) (bool, error) {
	addr := urlfmt.TrimHTTPPrefixes(origAddr)
	if addrSplits := strings.SplitN(addr, "/", 2); len(addrSplits) > 0 {
		addr = addrSplits[0]
	}

	addr, err := validateWithScheme(addr)
	if err != nil {
		return false, err
	}

	host, _, port, err := netutil.ParseEndpoint(addr)
	if err != nil {
		return false, err
	}
	if port == "" {
		if strings.HasPrefix(origAddr, "http://") {
			port = "80"
		} else {
			port = "443"
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	conn, err := proxy.AwareDialContextTLS(ctx, fmt.Sprintf("%s:%s", host, port), nil)
	if err != nil {
		switch err.(type) {
		case x509.CertificateInvalidError,
			x509.HostnameError,
			x509.UnknownAuthorityError,
			tls.RecordHeaderError,
			*tls.CertificateVerificationError:
			return false, nil
		}
		return false, err
	}
	utils.IgnoreError(conn.Close)
	return true, nil
}
