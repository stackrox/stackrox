package updater

import (
	"net/http"
	"regexp"
)

var releaseVersionPattern = regexp.MustCompile(`^v?([0-9]+\.[0-9]+\.[0-9]+)([-+].*)?$`)

// updaterUserAgent includes only the numeric release version, excluding build
// identifiers and prerelease suffixes. Unrecognized versions use "unknown".
func updaterUserAgent(buildVersion string) string {
	releaseVersion := "unknown"
	if match := releaseVersionPattern.FindStringSubmatch(buildVersion); match != nil {
		releaseVersion = match[1]
	}
	return "stackrox-updater/" + releaseVersion
}

type userAgentTransport struct {
	transport http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	out := req.Clone(req.Context())
	if out.Header == nil {
		out.Header = make(http.Header)
	}
	out.Header.Set("User-Agent", t.userAgent)
	return t.transport.RoundTrip(out)
}
