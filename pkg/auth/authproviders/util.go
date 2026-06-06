package authproviders

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// Some common user identity attribute keys.
const (
	GroupsAttribute = "groups"
	EmailAttribute  = "email"
	UseridAttribute = "userid"
	NameAttribute   = "name"
)

// missingPortErr is the error string returned by net.SplitHostPort when no port is present.
const missingPortErr = "missing port in address"

// NormalizeUIEndpoint validates and converts a UI endpoint to canonical host:port format.
// It strips URL schemes, defaults to port 443 when no port is given,
// and removes paths and trailing slashes.
func NormalizeUIEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errox.InvalidArgs.New("UI endpoint must not be empty")
	}

	// Ensure a scheme is present so url.Parse can extract the host correctly.
	withScheme := urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	hostPort := urlfmt.GetServerFromURL(withScheme)
	if hostPort == "" {
		return "", errox.InvalidArgs.Newf("UI endpoint %q is not a valid host[:port]", endpoint)
	}

	host, port, err := net.SplitHostPort(hostPort)
	switch {
	case err == nil && port == "":
		// Trailing colon with no port (e.g. "example.com:").
		return host + ":443", nil
	case err == nil:
		return hostPort, nil
	default:
		// "missing port in address" is the only expected error here — malformed
		// inputs like unbracketed IPv6 are already rejected by GetServerFromURL.
		var addrErr *net.AddrError
		if errors.As(err, &addrErr) && addrErr.Err == missingPortErr {
			return hostPort + ":443", nil
		}
		return "", errox.InvalidArgs.Newf("UI endpoint %q is not a valid host[:port]: %v", endpoint, err)
	}
}

// AllUIEndpoints returns all UI endpoints for a given auth provider, with the default UI endpoint first.
func AllUIEndpoints(providerProto *storage.AuthProvider) []string {
	if providerProto.GetUiEndpoint() == "" {
		return nil
	}
	return append([]string{providerProto.GetUiEndpoint()}, providerProto.GetExtraUiEndpoints()...)
}

// ExtractURLValuesFromRequest extracts url.Values from GET and POST requests.
func ExtractURLValuesFromRequest(r *http.Request) (url.Values, error) {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query(), nil
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			return nil, httputil.Errorf(http.StatusBadRequest, "could not parse form data: %v", err)
		}
		return r.Form, nil
	default:
		return nil, httputil.Errorf(http.StatusMethodNotAllowed, "method %s is not supported for this URL", r.Method)
	}
}
