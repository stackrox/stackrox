package httputil

import (
	"net/http"

	"github.com/pkg/errors"
)

var errTrafficDenied = errors.New("HTTP traffic denied")

// DenyTransport returns a [http.RoundTripper] which denies all requests.
func DenyTransport() http.RoundTripper {
	return RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		log.Errorf("Denied HTTP %s to %q", req.Method, req.URL.String())
		return nil, errTrafficDenied
	})
}
