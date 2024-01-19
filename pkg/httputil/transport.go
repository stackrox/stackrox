package httputil

import "net/http"

var _ http.RoundTripper = (*Transport)(nil)

type Transport struct {
	Default     http.RoundTripper
	Interceptor RoundTripInterceptor
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.Interceptor != nil {
		return t.Interceptor(req, t.Default.RoundTrip)
	}

	return t.Default.RoundTrip(req)
}
