package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/pkg/version"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// Service is the struct that manages the Metadata API
type serviceImpl struct{}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterMetadataServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterMetadataServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// GetMetadata returns the metadata for Rox.
func (s *serviceImpl) GetMetadata(context.Context, *v1.Empty) (*v1.Metadata, error) {
	return &v1.Metadata{
		Version:      version.GetMainVersion(),
		BuildFlavor:  buildinfo.BuildFlavor,
		ReleaseBuild: buildinfo.ReleaseBuild,
	}, nil
}
