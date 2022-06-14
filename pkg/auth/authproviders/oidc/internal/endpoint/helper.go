package endpoint

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/stringutils"
)

// ErrNoIssuerProvided is returned when issuer supplied to NewHelper is empty.
var ErrNoIssuerProvided = errors.New("no issuer provided")

// Helper encapsulates logic closely connected with issuer URL.
// It helps establish the canonical issuer URL, determine the correct HTTP client to use and adjust auth URL.
type Helper struct {
	parsedIssuer    url.URL
	canonicalIssuer string
	httpClient      *http.Client
	urlForDiscovery string
}

// Issuer returns a canonicalized issuer URL string.
func (h *Helper) Issuer() string {
	return h.canonicalIssuer
}

// URLsForDiscovery returns a slice of candidate issuer URLs.
// They are based on the canonicalized issuer URL string, with stripped query string and fragment,
// and removed "+insecure" scheme suffix, if any.
// Such URLs are suitable for passing to the go-oidc library to use for discovery.
// Currently the only difference between the (two) returned URLs is that one of them ends with a slash, and the other
// does not.
func (h *Helper) URLsForDiscovery() []string {
	var modifiedURL string
	if strings.HasSuffix(h.urlForDiscovery, "/") {
		modifiedURL = strings.TrimSuffix(h.urlForDiscovery, "/")
	} else {
		modifiedURL = h.urlForDiscovery + "/"
	}
	return []string{h.urlForDiscovery, modifiedURL}
}

// HTTPClient returns an *http.Client suitable for communication with the OIDC provider.
// This is a default client or a client with "InsecureSkipVerify: true" in its TLS config, depending on whether the
// issuer URL contained "+insecure" in its schema.
func (h *Helper) HTTPClient() *http.Client {
	return h.httpClient
}

// AdjustAuthURL optionally changes the auth URL endpoint to incorporate the query string and fragment from the issuer URL.
func (h *Helper) AdjustAuthURL(authEndpoint string) (string, error) {
	authURL, err := url.Parse(authEndpoint)
	if err != nil {
		return "", errors.Wrapf(err, "unparseable OAuth2 auth URL %q", authEndpoint)
	}

	authURL.RawQuery = stringutils.JoinNonEmpty("&", authURL.RawQuery, h.parsedIssuer.RawQuery)
	authURL.ForceQuery = authURL.ForceQuery || h.parsedIssuer.ForceQuery
	authURL.Fragment = stringutils.JoinNonEmpty("&", authURL.Fragment, h.parsedIssuer.Fragment)

	return authURL.String(), nil
}

// NewHelper takes the issuer URL string configured by the user.
// It returns an error if the issuer URL is empty, not parsable, or the scheme is "http".
// If the scheme is not specified, it prepends "https://".
func NewHelper(issuer string) (*Helper, error) {
	if issuer == "" {
		return nil, ErrNoIssuerProvided
	}

	if !strings.Contains(issuer, "://") {
		issuer = "https://" + issuer
	}

	issuerURL, err := url.Parse(issuer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse issuer URL")
	}

	if issuerURL.Scheme == "http" {
		return nil, errors.New("unencrypted http is not allowed for OIDC issuers")
	}

	urlForDiscovery := &url.URL{
		Opaque:  issuerURL.Opaque,
		Scheme:  issuerURL.Scheme,
		Host:    issuerURL.Host,
		Path:    issuerURL.Path,
		RawPath: issuerURL.RawPath,
	}

	var httpClient *http.Client
	if stringutils.ConsumeSuffix(&urlForDiscovery.Scheme, "+insecure") {
		httpClient = insecureHTTPClient
	} else {
		httpClient = http.DefaultClient
	}

	h := &Helper{
		parsedIssuer:    *issuerURL,
		canonicalIssuer: issuer,
		httpClient:      httpClient,
		urlForDiscovery: urlForDiscovery.String(),
	}
	return h, nil
}
