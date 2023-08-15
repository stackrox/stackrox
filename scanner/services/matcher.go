package services

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"github.com/stackrox/rox/scanner/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// matcherService represents a vulnerability matcher gRPC service.
type matcherService struct {
	v4.UnimplementedMatcherServer
	matcher matcher.Matcher
}

// NewMatcherService creates a new vulnerability matcher gRPC service.
func NewMatcherService(matcher matcher.Matcher) *matcherService {
	return &matcherService{
		matcher: matcher,
	}
}

func (s *matcherService) GetVulnerabilities(_ context.Context, _ *v4.GetVulnerabilitiesRequest) (*v4.VulnerabilityReport, error) {
	return nil, status.Error(codes.Unimplemented, "method GetVulnerabilities not implemented")
}

func (s *matcherService) GetMetadata(_ context.Context, _ *types.Empty) (*v4.Metadata, error) {
	return nil, status.Error(codes.Unimplemented, "method GetMetadata not implemented")
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *matcherService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterMatcherServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *matcherService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// TODO: Setup permissions for matcher.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *matcherService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
