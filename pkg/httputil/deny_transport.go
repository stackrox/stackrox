package httputil

import (
	"net/http"

	"github.com/pkg/errors"
)

var errDenied = errors.New("HTTP traffic denied")

var _ http.RoundTripper = (*denyTransport)(nil)

type denyTransport struct {}

// DenyTransport returns a [http.RoundTripper] which denies all requests.
func DenyTransport() http.RoundTripper {
	return denyTransport{}
}

func (denyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	log.Errorf("Denied HTTP %s to %q", req.Method, req.URL.String())
	return nil, errDenied
}
