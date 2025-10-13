package phonehome

import (
	"context"
	"net/http"
	"slices"

	"github.com/stackrox/rox/pkg/glob"
	"github.com/stackrox/rox/pkg/grpc/authn"
)

// NoHeaderOrAnyValue pattern allows no header or a header with any value.
const NoHeaderOrAnyValue glob.Pattern = ""

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

// HasHeader returns true if for each header pattern there is at least one
// request header with at least one matching value. A request without the
// expected header matches NoHeaderOrAnyValue pattern for this header.
func (rp *RequestParams) HasHeader(patterns map[string]glob.Pattern) bool {
	for header, expression := range patterns {
		if expression == NoHeaderOrAnyValue {
			continue
		}
		if rp.Headers == nil {
			return false
		}
		values := rp.Headers(header)
		if len(values) == 0 || !slices.ContainsFunc(values, expression.Match) {
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
