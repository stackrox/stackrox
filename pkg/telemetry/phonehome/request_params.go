package phonehome

import (
	"context"
	"net/http"

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

// PathMatches returns true if Path matches the glob pattern.
// E.g. path '/v1/object/id' matches pattern '*/object/*'.
func (rp *RequestParams) PathMatches(pattern Pattern) bool {
	return globCache[pattern].Match(rp.Path)
}

func hasValueMatching(values []string, pattern Pattern) bool {
	for _, value := range values {
		if globCache[pattern].Match(value) {
			return true
		}
	}
	return false
}

// HasHeader returns true if for each header pattern there is at least one
// matching value. A request without the expected header matches empty pattern
// for this header.
func (rp *RequestParams) HasHeader(patterns map[string]Pattern) bool {
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
