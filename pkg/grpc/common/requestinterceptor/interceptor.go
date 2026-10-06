package requestinterceptor

import (
	"context"
	"net/http"

	v1 "github.com/stackrox/rox/generated/api/v1"
	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
)

const userAgentHeaderKey = "User-Agent"

var log = logging.LoggerForModule()

// GetGRPCRequestDetails constructs a RequestParams for a gRPC invocation.
// For grpc-gateway requests it uses the HTTP method, path, status code, and
// headers, merging User-Agent values from both gRPC metadata and the HTTP
// request. For pure gRPC calls it uses the full method name as both Method
// and Path, derives the code from erroxGRPC.RoxErrorToGRPCCode, and builds
// Headers from gRPC metadata.
func GetGRPCRequestDetails(ctx context.Context, err error, grpcFullMethod string, req any) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil && grpcFullMethod != v1.PingService_Ping_FullMethodName {
		log.Debugf("Cannot identify user from context for method call %q: %v", grpcFullMethod, iderr)
	}

	ri := requestinfo.FromContext(ctx)

	// Use the wrapped HTTP request if provided by the grpc-gateway.
	if ri.HTTPRequest != nil {
		var path string
		if ri.HTTPRequest.URL != nil {
			path = ri.HTTPRequest.URL.Path
		}
		// Append the gRPC transport User-Agent from metadata to the
		// original HTTP headers so all User-Agent values are under one key.
		// The request has already been processed (we've got the result), so the
		// headers are ok to modify to avoid cloning.
		for _, ua := range ri.Metadata.Get(userAgentHeaderKey) {
			ri.HTTPRequest.Headers.Add(userAgentHeaderKey, ua)
		}
		return &RequestParams{
			UserID:  id,
			Method:  ri.HTTPRequest.Method,
			Path:    path,
			Code:    grpcError.ErrToHTTPStatus(err),
			GRPCReq: req,
			Headers: ri.HTTPRequest.Headers,
		}
	}

	return &RequestParams{
		UserID:  id,
		Method:  grpcFullMethod,
		Path:    grpcFullMethod,
		Code:    int(erroxGRPC.RoxErrorToGRPCCode(err)),
		GRPCReq: req,
		Headers: NewHeaders(ri.Metadata),
	}
}

// GetHTTPRequestDetails extracts the authenticated user (if any) from ctx and constructs
// a RequestParams describing the given HTTP request and response status.
func GetHTTPRequestDetails(ctx context.Context, r *http.Request, status int) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil {
		log.Debug("Cannot identify user from context: ", iderr)
	}

	return &RequestParams{
		UserID:  id,
		Method:  r.Method,
		Path:    r.URL.Path,
		Code:    status,
		HTTPReq: r,
		Headers: r.Header,
	}
}
