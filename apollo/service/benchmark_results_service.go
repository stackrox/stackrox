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

// NewBenchmarkResultsService returns the BenchmarkResultsService API.
func NewBenchmarkResultsService(storage db.Storage) *BenchmarkResultsService {
	return &BenchmarkResultsService{
		storage: storage,
	}
}

// BenchmarkResultsService is the struct that manages the benchmark API
type BenchmarkResultsService struct {
	storage db.BenchmarkResultsStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkResultsService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkResultsServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkResultsService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkResultsServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetBenchmarkResults retrieves benchmark results based on the request filters
func (s *BenchmarkResultsService) GetBenchmarkResults(ctx context.Context, request *v1.GetBenchmarkResultsRequest) (*v1.GetBenchmarkResultsResponse, error) {
	benchmarks, err := s.storage.GetBenchmarkResults(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarkResultsResponse{Benchmarks: benchmarks}, nil
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *BenchmarkResultsService) PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkResult) (*empty.Empty, error) {
	if err := s.storage.AddBenchmarkResult(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}
