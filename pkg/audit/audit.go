package audit

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	"google.golang.org/grpc"
)

// Auditor implements a unary server interceptor
type Auditor interface {
	UnaryServerInterceptor() func(context.Context, any, *grpc.UnaryServerInfo, grpc.UnaryHandler) (any, error)
	PostAuthHTTPInterceptor(handler http.Handler) http.Handler
	SendAuditMessage(ctx context.Context, req any, grpcMethod string, authError interceptor.AuthStatus, requestError error)
}
