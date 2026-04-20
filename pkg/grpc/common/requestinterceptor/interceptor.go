package requestinterceptor

import (
	"context"
	"net/http"
	"sync/atomic"

	v1 "github.com/stackrox/rox/generated/api/v1"
	erroxGRPC "github.com/stackrox/rox/pkg/errox/grpc"
	"github.com/stackrox/rox/pkg/grpc/authn"
	grpcError "github.com/stackrox/rox/pkg/grpc/errors"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"google.golang.org/grpc"
)

const userAgentHeaderKey = "User-Agent"

var log = logging.LoggerForModule()

// RequestHandler is called with the computed RequestParams for every API
// request. Handlers are registered with the RequestInterceptor.
type RequestHandler func(*RequestParams)

// RequestInterceptor computes RequestParams once per API request and fans
// out to all registered handlers. If no handlers are registered, the
// interceptor is a no-op and does not compute RequestParams.
type RequestInterceptor struct {
	handlers sync.Map
	count    atomic.Int32
}

// NewRequestInterceptor creates a new interceptor registry.
func NewRequestInterceptor() *RequestInterceptor {
	return &RequestInterceptor{}
}

// Add registers a named handler. Replaces any existing handler with the
// same name.
func (ri *RequestInterceptor) Add(name string, h RequestHandler) {
	if _, loaded := ri.handlers.Swap(name, h); !loaded {
		ri.count.Add(1)
	}
}

// Remove unregisters a handler by name.
func (ri *RequestInterceptor) Remove(name string) {
	if _, loaded := ri.handlers.LoadAndDelete(name); loaded {
		ri.count.Add(-1)
	}
}

func (ri *RequestInterceptor) dispatch(rp *RequestParams) {
	ri.handlers.Range(func(_, v any) bool {
		v.(RequestHandler)(rp)
		return true
	})
}

func (ri *RequestInterceptor) hasHandlers() bool {
	return ri.count.Load() > 0
}

// UnaryServerInterceptor returns a gRPC unary server interceptor.
func (ri *RequestInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if ri.hasHandlers() {
			rp := getGRPCRequestDetails(ctx, err, info.FullMethod, req)
			ri.dispatch(rp)
		}
		return resp, err
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor.
func (ri *RequestInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if ri.hasHandlers() {
			rp := getGRPCRequestDetails(ss.Context(), err, info.FullMethod, nil)
			ri.dispatch(rp)
		}
		return err
	}
}

// HTTPInterceptor returns an HTTP middleware interceptor.
func (ri *RequestInterceptor) HTTPInterceptor() httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrappedWriter := httputil.NewStatusTrackingWriter(w)
			handler.ServeHTTP(wrappedWriter, r)
			if ri.hasHandlers() {
				status := http.StatusOK
				if sptr := wrappedWriter.GetStatusCode(); sptr != nil {
					status = *sptr
				}
				rp := getHTTPRequestDetails(r.Context(), r, status)
				ri.dispatch(rp)
			}
		})
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
		Headers: r.Header,
	}
}
