package interceptor

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authContextKey struct{}

// AuthStatus is a wrapper around an authentication error
// It is used to distinguish between an unset error and a nil error
type AuthStatus struct {
	Error error
}

func (a *AuthStatus) String() string {
	if a.Error == nil {
		return ""
	}
	return a.Error.Error()
}

// AuthContextUpdaterInterceptor returns a new unary server interceptors that performs per-request auth.
func AuthContextUpdaterInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var newCtx context.Context
		var err error
		if overrideSrv, ok := info.Server.(grpc_auth.ServiceAuthFuncOverride); ok {
			newCtx, err = overrideSrv.AuthFuncOverride(ctx, info.FullMethod)
		} else {
			newCtx, err = deny.AuthFunc(ctx)
		}
		newCtx = context.WithValue(newCtx, authContextKey{}, AuthStatus{Error: err})
		return handler(newCtx, req)
	}
}

// GetAuthErrorFromContext returns the auth error from the context
func GetAuthErrorFromContext(ctx context.Context) AuthStatus {
	authCtxValue := ctx.Value(authContextKey{})
	if authCtxValue == nil {
		return AuthStatus{Error: status.Error(codes.Internal, "authentication status is always required")}
	}
	return authCtxValue.(AuthStatus)
}

// AuthCheckerInterceptor actually checks the auth and rejects if it had an error
func AuthCheckerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if authStatus := GetAuthErrorFromContext(ctx); authStatus.Error != nil {
			return nil, authStatus.Error
		}
		return handler(ctx, req)
	}
}
