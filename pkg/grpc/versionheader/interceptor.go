package versionheader

import (
	"context"

	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryServerInterceptor returns a gRPC unary server interceptor that sets the
// Central version in the response metadata for authenticated requests.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		setVersionHeader(ctx)
		return handler(ctx, req)
	}
}

func setVersionHeader(ctx context.Context) {
	if authn.IdentityFromContextOrNil(ctx) == nil {
		return
	}
	_ = grpc.SetHeader(ctx, metadata.Pairs(clientconn.CentralVersionHeader, version.GetMainVersion()))
}
