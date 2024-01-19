package httputil

import (
	"net/http"
	"strings"

	"github.com/stackrox/rox/pkg/httputil"
)

// MuxTransport returns a [http.RoundTripper] which handles HTTP requests as follows:
//
//   - If the request is destined for Central, use the given Central [http.RoundTripper].
//   - If the request is destined for Sensor, use the given Sensor [http.RoundTripper].
//   - Otherwise, use the default [http.RoundTripper].
//
// This is designed for Scanner's specific use-case. It is possible to generalize this, but that's not necessary
// at this time.
func MuxTransport(centralTransport, sensorTransport, defaultTransport http.RoundTripper) http.RoundTripper {
	return httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case strings.HasPrefix(req.URL.Host, "central"):
			return centralTransport.RoundTrip(req)
		case strings.HasPrefix(req.URL.Host, "sensor"):
			return sensorTransport.RoundTrip(req)
		default:
			return defaultTransport.RoundTrip(req)
		}
	})
}
