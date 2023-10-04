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

// CheckTLS checks if the address is using TLS
func CheckTLS(ctx context.Context, origAddr string) (bool, error) {
	addr := urlfmt.TrimHTTPPrefixes(origAddr)
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
		// TODO: revert Go 1.20 change for https://github.com/stackrox/stackrox/commit/5b163bd264197c379acb3b3be63c5e344ab48793#diff-7bc8351b84225aaf8f3f66811f6870eded759036ff293085de708860c89e149a.
		case x509.CertificateInvalidError, x509.HostnameError, x509.UnknownAuthorityError, tls.RecordHeaderError:
			return false, nil
		}
		return false, err
	}
	utils.IgnoreError(conn.Close)
	return true, nil
}
