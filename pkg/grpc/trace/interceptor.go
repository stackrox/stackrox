package trace

import (
	"context"

	"github.com/stackrox/rox/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

var log = logging.LoggerForModule()

// Trace contains service identity information.
type Trace struct {
	ServiceName string
}

// TraceOption is the option function for the trace configuration.
type TraceOption func(t *Trace)

// WithServiceName adds the service name to the trace.
func WithServiceName(name string) TraceOption {
	return func(t *Trace) {
		t.ServiceName = name
	}
}

func applyTraceOptions(options ...TraceOption) *Trace {
	trace := &Trace{}
	for _, opt := range options {
		opt(trace)
	}
	return trace
}

func IncomingTraceInterceptor(options ...TraceOption) grpc.UnaryServerInterceptor {
	trace := applyTraceOptions(options...)
	return func(
		ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler,
	) (interface{}, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		md.Append(logging.ServiceNameContextValue, trace.ServiceName)
		return handler(ctx, req)
	}
}
