package phonehome

import (
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authn"
)

// RequestParams holds intercepted call parameters.
type RequestParams struct {
	UserAgent string
	UserID    authn.Identity
	Path      string
	Code      int
	GRPCReq   any
	HTTPReq   *http.Request
}

// GetMethod returns the HTTP method for HTTP requests, or rp.Path otherwise.
func (rp *RequestParams) GetMethod() string {
	switch {
	case rp.HTTPReq == nil:
		return rp.Path // i.e. in the form of /service/method for gRPC requests.
	case rp.HTTPReq.Method != "":
		return rp.HTTPReq.Method
	default:
		return http.MethodGet
	}
}
