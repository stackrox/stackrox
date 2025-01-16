package phonehome

import (
	"context"
	"net/http"

	"github.com/gobwas/glob"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

const NoHeaderOrAnyValuePattern = "*"

// RequestParams holds intercepted call parameters.
type RequestParams struct {
	UserID  authn.Identity
	Method  string
	Path    string
	Code    int
	GRPCReq any
	HTTPReq *http.Request
	// HTTP Headers or, for pure gRPC, the metadata. Includes the User-Agent.
	Headers func(string) []string
}

// ServiceMethod describes a service method with its gRPC and HTTP variants.
type ServiceMethod struct {
	GRPCMethod string
	HTTPMethod string
	HTTPPath   string
}

// PathMatches returns true if Path matches the glob pattern.
// E.g. path '/v1/object/id' matches pattern '*/object/*'.
func (rp *RequestParams) PathMatches(pattern string) bool {
	return glob.MustCompile(pattern).Match(rp.Path)
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

func hasValueMatching(values []string, expression string) bool {
	for _, value := range values {
		if glob.MustCompile(expression).Match(value) {
			return true
		}
	}
	return false
}

// HasHeader returns true if for each header pattern there is at least one
// matching value. A request without the expected header matches empty pattern
// for this header.
func (rp *RequestParams) HasHeader(patterns map[string]string) bool {
	for header, expression := range patterns {
		if expression == NoHeaderOrAnyValuePattern {
			continue
		}
		if rp.Headers == nil {
			return false
		}
		values := rp.Headers(header)
		if len(values) == 0 || !hasValueMatching(values, expression) {
			return false
		}
	}
	return true
}

// Is checks wether the request targets the service method: either gRPC or HTTP.
func (rp *RequestParams) Is(s *ServiceMethod) bool {
	return rp.Method == s.GRPCMethod || (rp.Method == s.HTTPMethod && rp.PathMatches(s.HTTPPath))
}

// GetGRPCRequestBody returns the request body with the type inferred from the
// API handler provided as the first argument.
func GetGRPCRequestBody[
	F func(Service, context.Context, *Request) (*Response, error),
	Service any,
	Request any,
	Response any,
](_ F, rp *RequestParams) *Request {
	if body, ok := rp.GRPCReq.(*Request); ok {
		return body
	}
	return nil
}
