package phonehome

import (
	"context"
	"net/http"

	v1 "github.com/stackrox/rox/generated/api/v1"
	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/telemetry/phonehome/telemeter"
)

const userAgentHeaderKey = "User-Agent"

// Interceptor is a function which will be called on every API call if none of
// the previous interceptors in the chain returned false.
// An Interceptor function may add custom properties to the props map so that
// they appear in the event.
type Interceptor func(rp *RequestParams, props map[string]any) bool

func (c *Client) track(rp *RequestParams) {
	if !c.IsActive() {
		return
	}
	c.interceptorsLock.RLock()
	defer c.interceptorsLock.RUnlock()
	if len(c.interceptors) == 0 {
		return
	}
	opts := append(c.WithGroups(),
		telemeter.WithUserID(c.HashUserAuthID(rp.UserID)))
	t := c.Telemeter()
	for event, funcs := range c.interceptors {
		props := map[string]any{}
		ok := true
		for _, interceptor := range funcs {
			if ok = interceptor(rp, props); !ok {
				break
			}
		}
		if ok {
			t.Track(event, props, opts...)
		}
	}
}

// getGRPCRequestDetails constructs a RequestParams for a gRPC invocation.
// For grpc-gateway requests it uses the HTTP method, path, status code, and
// headers, merging User-Agent values from both gRPC metadata and the HTTP
// request. For pure gRPC calls it uses the full method name as both Method
// and Path, derives the code from erroxGRPC.RoxErrorToGRPCCode, and builds
// Headers from gRPC metadata.
func getGRPCRequestDetails(ctx context.Context, err error, grpcFullMethod string, req any) *RequestParams {
	id, iderr := authn.IdentityFromContext(ctx)
	if iderr != nil && grpcFullMethod != v1.PingService_Ping_FullMethodName { // Ignore readiness probes.
		log.Debugf("Cannot identify user from context for method call %q: %v", grpcFullMethod, iderr)
	}

	ri := requestinfo.FromContext(ctx)

	// Use the wrapped HTTP request if provided by the grpc-gateway.
	if ri.HTTPRequest != nil {
		var path string
		if ri.HTTPRequest.URL != nil {
			path = ri.HTTPRequest.URL.Path
		}
		// Override the User-Agent with the gRPC client or the grpc-gateway user
		// agent.
		grpcClientAgent := ri.Metadata.Get(userAgentHeaderKey)
		if clientAgent := ri.HTTPRequest.Headers.Get(userAgentHeaderKey); clientAgent != "" {
			grpcClientAgent = append(grpcClientAgent, clientAgent)
		}
		header := Headers(ri.HTTPRequest.Headers)
		// The request has already been processed (we've got the result), so the
		// headers are ok to modify to avoid cloning.
		header.Set(userAgentHeaderKey, grpcClientAgent...)
		return &RequestParams{
			UserID:  id,
			Method:  ri.HTTPRequest.Method,
			Path:    path,
			Code:    grpcError.ErrToHTTPStatus(err),
			GRPCReq: req,
			Headers: header,
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

// getHTTPRequestDetails extracts the authenticated user (if any) from ctx and constructs
// a RequestParams describing the given HTTP request and response status.
// If user identity cannot be obtained, a debug message is logged.
// The returned RequestParams contains the request method, URL path, provided status code,
// the original *http.Request (HTTPReq) and a Headers wrapper created from r.Header.
func getHTTPRequestDetails(ctx context.Context, r *http.Request, status int) *RequestParams {
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
		Headers: Headers(r.Header),
	}
}
