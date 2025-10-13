package httputil

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	errTrafficDenied = errors.New("HTTP traffic denied")

	// DenyTransport returns a http.RoundTripper which denies all requests.
	DenyTransport http.RoundTripper = httputil.RoundTripperFunc(denyTransport)
)

func denyTransport(req *http.Request) (*http.Response, error) {
	zlog.Error(context.Background()).
		Str("method", req.Method).
		Str("url", req.URL.String()).
		Msg("denied HTTP request")
	return nil, utils.ShouldErr(errTrafficDenied)
}
