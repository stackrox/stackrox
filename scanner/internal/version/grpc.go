package version

import (
	"context"

	"github.com/stackrox/rox/pkg/scannerv4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md := metadata.Pairs(scannerv4.ServiceVersionHeader, Version)
		if err := grpc.SetHeader(ctx, md); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}
}
