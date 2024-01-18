package httputil

import (
	"net/http"
	"strings"
)

var _ http.RoundTripper = (*muxTransport)(nil)

type muxTransport struct {
	centralTransport http.RoundTripper
	sensorTransport  http.RoundTripper
	defaultTransport http.RoundTripper
}

// MuxTransport returns a [http.RoundTripper] which handles HTTP as follows:
//
//   * If the request is destined for Central, use the given Central [http.RoundTripper].
//   * If the request is destined for Sensor, use the given Sensor [http.RoundTripper].
//   * Otherwise, use the default [http.RoundTripper].
func MuxTransport(centralTransport, sensorTransport, defaultTransport http.RoundTripper) http.RoundTripper {
	return &muxTransport{
		centralTransport: centralTransport,
		sensorTransport:  sensorTransport,
		defaultTransport: defaultTransport,
	}
}

func (r *muxTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case strings.HasPrefix(req.URL.Host, "central"):
		return r.centralTransport.RoundTrip(req)
	case strings.HasPrefix(req.URL.Host, "sensor"):
		return r.sensorTransport.RoundTrip(req)
	default:
		return r.defaultTransport.RoundTrip(req)
	}
}
