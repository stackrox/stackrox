package phonehome

import (
	"io"
	"net/http"
	"strings"

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

// PathMatches returns true if Path equals to pattern or matches '*'-terminating
// wildcard. E.g. path '/v1/object/id' will match pattern '/v1/object/*'.
func (rp *RequestParams) PathMatches(pattern string) bool {
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(rp.Path, pattern[0:len(pattern)-1])
	}
	return rp.Path == pattern
}

// HasPathIn returns true if Path matches an element in patterns.
func (rp *RequestParams) HasPathIn(patterns []string) bool {
	for _, p := range patterns {
		if rp.PathMatches(p) {
			return true
		}
	}
	return false
}

// Is checks wether the request targets the service method: either gRPC or HTTP.
func (rp *RequestParams) Is(s *ServiceMethod) bool {
	return rp.Method == s.GRPCMethod || (rp.Method == s.HTTPMethod && rp.PathMatches(s.HTTPPath))
}

// GetGRPCRequestBody sets the output body argument. Returns io.EOF on empty body.
func GetGRPCRequestBody[T any](rp *RequestParams, body **T) error {
	var ok bool
	if *body, ok = rp.GRPCReq.(*T); !ok || *body == nil {
		return io.EOF
	}
	return nil
}
