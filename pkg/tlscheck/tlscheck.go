package tlscheck

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/stackrox/rox/pkg/netutil"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	timeout = 500 * time.Millisecond
)

// CheckTLS checks if the address is using TLS
func CheckTLS(origAddr string) (bool, error) {
	addr := urlfmt.TrimHTTPPrefixes(origAddr)
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

	dialer := &net.Dialer{
		Timeout: timeout,
	}
	conn, err := tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:%s", host, port), nil)
	if err != nil {
		switch err.(type) {
		case x509.CertificateInvalidError, x509.HostnameError, x509.UnknownAuthorityError, tls.RecordHeaderError:
			return false, nil
		}
		return false, err
	}
	utils.IgnoreError(conn.Close)
	return true, nil
}
