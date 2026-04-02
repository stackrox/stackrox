package phonehome

import (
	"context"
	"net/http"

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
	Headers Headers
}

type GlobMap map[string]glob.Pattern

// MatchHeaders checks whether the request headers satisfy all given patterns.
// Returns nil if any pattern fails to match or if headers are absent. Returns
// non-nil (possibly empty) Headers containing only the matched values on
// success. Absent headers satisfy NoHeaderOrAnyValue without appearing in the
// result.
func (rp *RequestParams) MatchHeaders(patterns GlobMap) Headers {
	result := make(Headers)
	for header, expression := range patterns {
		values := rp.Headers.GetMatching(header, expression)
		if values == nil {
			if expression != NoHeaderOrAnyValue {
				return nil
			}
			continue
		}
		result[header] = values
	}
	return result
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
