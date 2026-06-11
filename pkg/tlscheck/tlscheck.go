package tlscheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
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

func addrValid(addr string) error {
	if strings.Contains(addr, "://") {
		return fmt.Errorf("URL %q should not contain scheme prefix", addr)
	}
	if strings.ContainsAny(addr, " \t\n\r") {
		return fmt.Errorf("URL %q contains illegal whitespace characters", addr)
	}
	// Bare IPv6 addresses must use bracketed format [IPv6]:port (RFC 2732).
	hostPart := strings.SplitN(addr, "/", 2)[0]
	if !strings.HasPrefix(hostPart, "[") && strings.Count(hostPart, ":") > 1 {
		return fmt.Errorf("bare IPv6 address in %q: use [IPv6]:port format", hostPart)
	}
	return nil
}

// CheckTLS checks if the address is using TLS
func CheckTLS(ctx context.Context, origAddr string) (bool, error) {
	addr := urlfmt.TrimHTTPPrefixes(origAddr)
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
