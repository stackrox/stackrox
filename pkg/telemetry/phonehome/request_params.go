package phonehome

import (
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authn"
)

// RequestParams holds intercepted call parameters.
type RequestParams struct {
	UserAgent string
	UserID    authn.Identity
	Method    string
	Path      string
	Code      int
	GRPCReq   any
	HTTPReq   *http.Request
}

// ServiceMethod describes a service method with its gRPC and HTTP variants.
type ServiceMethod struct {
	GRPCMethod string
	HTTPMethod string
	HTTPPath   string
}

// Is checks wether the request targets the service method: either gRPC or HTTP.
func (rp *RequestParams) Is(s *ServiceMethod) bool {
	return rp.Method == s.GRPCMethod || (rp.Method == s.HTTPMethod && rp.Path == s.HTTPPath)
}
