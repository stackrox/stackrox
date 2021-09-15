package observe

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/timestamp"
	"google.golang.org/grpc"
)

// TODO(ROX-7951): Support non-gRPC requests as well, e.g., `/api/graphql`.

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

func sendAuthzTrace(ctx context.Context, authzTraceSink AuthzTraceSink, rpcMethod string, handlerErr error, trace *AuthzTrace) {
	traceResp := &v1.AuthorizationTraceResponse{
		ArrivedAt:   trace.arrivedAt.LoadAtomic().GogoProtobuf(),
		ProcessedAt: timestamp.Now().GogoProtobuf(),
		Request:     calculateRequest(ctx, rpcMethod),
		Response:    calculateResponse(handlerErr),
		User:        calculateUser(ctx),
		Trace:       calculateTrace(trace),
	}
	authzTraceSink.PublishAuthzTrace(traceResp)
}
