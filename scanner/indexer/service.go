package indexer

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// indexerService represents an image indexer gRPC service.
type indexerService struct {
	v4.UnimplementedIndexerServer
	indexer *Indexer
}

// NewIndexerService creates a new image indexer gRPC service.
func NewIndexerService(indexer *Indexer) (*indexerService, error) {
	return &indexerService{
		indexer: indexer,
	}, nil
}

func (s *indexerService) CreateIndexReport(_ context.Context, _ *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	return nil, status.Error(codes.Unimplemented, "method CreateIndexReport not implemented")
}
func (s *indexerService) GetIndexReport(_ context.Context, _ *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	return nil, status.Error(codes.Unimplemented, "method GetIndexReport not implemented")
}
func (s *indexerService) HasIndexReport(_ context.Context, _ *v4.HasIndexReportRequest) (*types.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "method HasIndexReport not implemented")
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *indexerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterIndexerServer(grpcServer, s)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *indexerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// TODO: Setup permissions for indexer.
	return ctx, allow.Anonymous().Authorized(ctx, fullMethodName)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *indexerService) RegisterServiceHandler(_ context.Context, _ *runtime.ServeMux, _ *grpc.ClientConn) error {
	// Currently we do not set up gRPC gateway for the matcher.
	return nil
}
