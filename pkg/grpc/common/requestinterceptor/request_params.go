package requestinterceptor

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authn"
	"google.golang.org/grpc/metadata"
)

// RequestParams holds intercepted call parameters.
type RequestParams struct {
	UserID  authn.Identity
	Method  string
	Path    string
	Code    int
	GRPCReq any
	HTTPReq *http.Request
	// HTTP Headers or, for pure gRPC, the metadata. Includes the User-Agent.
	Headers http.Header
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

// NewHeaders creates http.Header from gRPC metadata, canonicalizing the
// lowercase keys used by metadata.MD into the format expected by http.Header.
func NewHeaders(m metadata.MD) http.Header {
	h := make(http.Header, len(m))
	for k, vs := range m {
		for _, v := range vs {
			h.Add(k, v)
		}
	}
	return h
}
