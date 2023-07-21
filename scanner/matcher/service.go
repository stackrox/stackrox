package matcher

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type matcherService struct {
	v4.UnimplementedMatcherServer
	matcher *Matcher
}

func NewMatcherService(matcher *Matcher) (*matcherService, error) {
	return &matcherService{
		matcher: matcher,
	}, nil
}

func (s *matcherService) GetVulnerabilities(ctx context.Context, req *v4.GetVulnerabilitiesRequest) (*v4.VulnerabilityReport, error) {
	return nil, status.Error(codes.Unimplemented, "method GetVulnerabilities not implemented")
}

func (s *matcherService) GetMetadata(ctx context.Context, req *types.Empty) (*v4.Metadata, error) {
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
func (s *matcherService) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
