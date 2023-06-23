package indexer

import (
	"context"

	"github.com/gogo/protobuf/types"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type indexerService struct {
	v4.UnimplementedIndexerServer
	indexer *Indexer
}

func NewIndexerService(indexer *Indexer) (*indexerService, error) {
	return &indexerService{
		indexer: indexer,
	}, nil
}

func (s *indexerService) CreateIndexReport(ctx context.Context, req *v4.CreateIndexReportRequest) (*v4.IndexReport, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateIndexReport not implemented")
}
func (s *indexerService) GetIndexReport(ctx context.Context, req *v4.GetIndexReportRequest) (*v4.IndexReport, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetIndexReport not implemented")
}
func (s *indexerService) HasIndexReport(ctx context.Context, req *v4.HasIndexReportRequest) (*types.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HasIndexReport not implemented")
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *indexerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v4.RegisterIndexerServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *indexerService) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return nil
}
