package observe

import (
	"context"
	"net/http"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/protoconv"
	"github.com/stackrox/rox/pkg/timestamp"
	"google.golang.org/grpc"
)

// AuthzTraceInterceptor supports tracing for authorization decisions by
// extracting an instance of a specific struct from the context which was
// (hopefully) filled in by authorizers as they made authorization decisions.
func AuthzTraceInterceptor(authzTraceSink AuthzTraceSink) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)

		if trace := AuthzTraceFromContext(ctx); trace != nil {
			go sendAuthzTrace(ctx, authzTraceSink, info.FullMethod, err, trace)
		}

		return resp, err
	}
}

// AuthzTraceHTTPInterceptor serves as AuthzTraceInterceptor for non-GRPC requests.
func AuthzTraceHTTPInterceptor(authzTraceSink AuthzTraceSink) httputil.HTTPInterceptor {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			statusTrackingWriter := httputil.NewStatusTrackingWriter(w)
			handler.ServeHTTP(statusTrackingWriter, r)
			if trace := AuthzTraceFromContext(r.Context()); trace != nil {
				err := statusTrackingWriter.GetStatusCodeError()
				go sendAuthzTrace(r.Context(), authzTraceSink, "", err, trace)
			}
		})
	}
}

func sendAuthzTrace(ctx context.Context, authzTraceSink AuthzTraceSink, rpcMethod string, handlerErr error, trace *AuthzTrace) {
	traceResp := &v1.AuthorizationTraceResponse{}
	traceResp.SetArrivedAt(protoconv.ConvertMicroTSToProtobufTS(trace.arrivedAt.LoadAtomic()))
	traceResp.SetProcessedAt(protoconv.ConvertMicroTSToProtobufTS(timestamp.Now()))
	traceResp.SetRequest(calculateRequest(ctx, rpcMethod))
	traceResp.SetResponse(calculateResponse(handlerErr))
	traceResp.SetUser(calculateUser(ctx))
	traceResp.SetTrace(calculateTrace(trace))
	authzTraceSink.PublishAuthzTrace(traceResp)
}
