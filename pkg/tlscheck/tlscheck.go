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

// addrValid validates the URL.
// It returns an error if addr contains scheme prefix or illegal characters.
func addrValid(addr string) error {
	if strings.Contains(addr, "://") {
		return fmt.Errorf("URL %q should not contain scheme prefix", addr)
	}

	// Check for illegal characters (spaces, tabs, newlines)
	if strings.ContainsAny(addr, " \t\n\r") {
		return fmt.Errorf("URL %q contains illegal whitespace characters", addr)
	}

	// Extract host part (before any path)
	hostPart := strings.SplitN(addr, "/", 2)[0]

	// For IPv6 addresses, wrap in brackets for url.Parse validation
	// This handles cases like "1::" or "2001:...:8329" but NOT "IPv6:port" format
	// The "IPv6:port" format (e.g., "2001:...:8329:61273") is ambiguous but accepted
	// per RFC2732 interpretation - we skip bracketing for simplicity
	if strings.Count(hostPart, ":") > 1 && !strings.HasPrefix(hostPart, "[") {
		// Try to use netutil.ParseEndpoint to validate it's a legitimate address
		// This accepts IPv6 and IPv6:port formats
		_, _, _, err := netutil.ParseEndpoint(hostPart)
		if err != nil {
			return err
		}
		// Accept it without url.Parse validation (which would reject unbracketed IPv6)
		return nil
	}

	// For non-IPv6 addresses, use url.Parse for validation
	_, err := url.Parse("https://" + addr)
	return err
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
