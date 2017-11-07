package service

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkService returns the BenchmarkService API.
func NewBenchmarkService(storage db.Storage) *BenchmarkService {
	return &BenchmarkService{
		storage: storage,
	}
}

// BenchmarkService is the struct that manages the benchmark API
type BenchmarkService struct {
	storage db.Storage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetBenchmark retrieves a benchmark passed on the request filters
func (s *BenchmarkService) GetBenchmark(ctx context.Context, request *v1.GetBenchmarkRequest) (*v1.GetBenchmarkResponse, error) {
	log.Infof("%+v", request)
	return &v1.GetBenchmarkResponse{}, status.Error(codes.Unimplemented, "Not implemented")
}

// PostBenchmark inserts a new benchmark into the system
func (s *BenchmarkService) PostBenchmark(ctx context.Context, request *v1.BenchmarkPayload) (*empty.Empty, error) {
	log.Infof("%+v", request)
	return &empty.Empty{}, status.Error(codes.Unimplemented, "Not implemented")
}
