package phonehome

import (
	"net/http"
	"strings"

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

// GetProtocol returns HTTP for requests with HTTP request and gRPC otherwise.
func (rp *RequestParams) GetProtocol() string {
	if rp.HTTPReq != nil {
		return "HTTP"
	}
	return "gRPC"
}

// GetMethod returns the HTTP method for HTTP requests, or the method matching
// the API path prefix for gRPC requests. Default: GET.
func (rp *RequestParams) GetMethod() string {
	if rp.HTTPReq != nil {
		if rp.HTTPReq.Method == "" {
			return http.MethodGet
		}
	} else {
		path := rp.Path[strings.LastIndex(rp.Path, "/")+1:]
		switch {
		case strings.HasPrefix(path, "Get"):
			return http.MethodGet
		case strings.HasPrefix(path, "Post"):
			return http.MethodPost
		case strings.HasPrefix(path, "Put"):
			return http.MethodPut
		case strings.HasPrefix(path, "Delete"):
			return http.MethodDelete
		case strings.HasPrefix(path, "Patch"):
			return http.MethodPatch
		case strings.HasPrefix(path, "Head"):
			return http.MethodHead
		case strings.HasPrefix(path, "Connect"):
			return http.MethodConnect
		case strings.HasPrefix(path, "Options"):
			return http.MethodOptions
		case strings.HasPrefix(path, "Trace"):
			return http.MethodTrace
		}
	}
	return http.MethodGet
}
