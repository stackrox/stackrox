package httputil

import (
	"context"
	"net/http"

	"github.com/pkg/errors"
	"github.com/quay/zlog"
	"github.com/stackrox/rox/pkg/httputil"
)

var errTrafficDenied = errors.New("HTTP traffic denied")

// DenyTransport returns a [http.RoundTripper] which denies all requests.
func DenyTransport(ctx context.Context) http.RoundTripper {
	return httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		zlog.Error(ctx).
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Msg("denied HTTP request")
		return nil, errTrafficDenied
	})
}
