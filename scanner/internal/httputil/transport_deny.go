package httputil

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	errTrafficDenied = errors.New("HTTP traffic denied")

	// DenyTransport returns a http.RoundTripper which denies all requests.
	DenyTransport http.RoundTripper = httputil.RoundTripperFunc(denyTransport)
)

func denyTransport(req *http.Request) (*http.Response, error) {
	slog.ErrorContext(context.Background(), "denied HTTP request", "method", req.Method, "url", req.URL.String())
	return nil, utils.ShouldErr(errTrafficDenied)
}
