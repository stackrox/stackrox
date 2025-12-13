package apirequestlog

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/central/metrics/custom/api_requests"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/telemetry/phonehome"
	"google.golang.org/grpc"
)

// UnaryServerInterceptor creates a gRPC unary interceptor that tracks API
// request metadata.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		rp := phonehome.GetGRPCRequestDetails(ctx, err, info.FullMethod, req)
		api_requests.RecordRequest(rp)
		return resp, err
	}
}

// StreamServerInterceptor creates a gRPC stream interceptor that tracks API
// request metadata.
func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		rp := phonehome.GetGRPCRequestDetails(ss.Context(), err,
			info.FullMethod, nil)
		api_requests.RecordRequest(rp)
		return err
	}
}

// HTTPInterceptor creates an HTTP interceptor that tracks API request metadata.
func HTTPInterceptor() httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wrappedWriter := httputil.NewStatusTrackingWriter(w)
			handler.ServeHTTP(wrappedWriter, r)
			statusCode := 0
			if statusCodePtr := wrappedWriter.GetStatusCode(); statusCodePtr != nil {
				statusCode = *statusCodePtr
			}
			rp := phonehome.GetHTTPRequestDetails(r.Context(), r, statusCode)
			api_requests.RecordRequest(rp)
		})
	}
}
