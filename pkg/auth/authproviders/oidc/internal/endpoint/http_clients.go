package endpoint

import (
	"github.com/stackrox/rox/pkg/auth/authproviders/idputil"
)

var (
	// defaultHTTPClient is a proxy-aware HTTP client for secure OIDC connections.
	// It includes a timeout to prevent hanging calls to external IdPs.
	defaultHTTPClient = idputil.NewHTTPClient()

	// insecureHTTPClient is a proxy-aware HTTP client that skips TLS verification.
	// This is used when the issuer URL contains the "+insecure" scheme suffix.
	// It includes a timeout to prevent hanging calls to external IdPs.
	insecureHTTPClient = idputil.NewInsecureHTTPClient()
)
