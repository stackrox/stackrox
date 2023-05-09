package audit

import (
	"context"
	"net/http"

	"github.com/stackrox/rox/pkg/grpc/authz/interceptor"
	"google.golang.org/grpc"
)

// Auditor implements a unary server interceptor
type Auditor interface {
	UnaryServerInterceptor() func(context.Context, interface{}, *grpc.UnaryServerInfo, grpc.UnaryHandler) (interface{}, error)
	HTTPInterceptor(handler http.Handler) http.Handler
	SendAdhocAuditMessage(ctx context.Context, req interface{}, grpcMethod string, authError interceptor.AuthStatus, requestError error)
}
