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

// addrValid validates the URL. It assumes scheme prefixes have been removed
func addrValid(addr string) error {
	addrNoScheme := urlfmt.TrimHTTPPrefixes(addr)
	// url.Parse requires scheme to trigger the correct variant of parsing (it has two)
	_, err := url.Parse("https://" + addrNoScheme)
	return err
}

// CheckTLS checks if the address is using TLS
func CheckTLS(ctx context.Context, origAddr string) (bool, error) {
	addr := urlfmt.TrimHTTPPrefixes(origAddr)
	// Ellimitate obvious mistakes in the host name
	addr = strings.TrimSpace(addr)
	if err := addrValid(addr); err != nil {
		return false, err
	}

	if addrSplits := strings.SplitN(addr, "/", 2); len(addrSplits) > 0 {
		addr = addrSplits[0]
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
