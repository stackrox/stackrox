package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewBenchmarkService returns the BenchmarkService API.
func NewBenchmarkService(storage db.BenchmarkStorage) *BenchmarkService {
	return &BenchmarkService{
		storage: storage,
	}
}

// BenchmarkService is the struct that manages the benchmark API
type BenchmarkService struct {
	storage db.BenchmarkStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *BenchmarkService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// GetBenchmark returns the benchmark by the passed name
func (s *BenchmarkService) GetBenchmark(ctx context.Context, request *v1.ResourceByID) (*v1.Benchmark, error) {
	benchmark, exists, err := s.storage.GetBenchmark(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Benchmark with id %v is not found", request.GetId()))
	}
	return benchmark, nil
}

// GetChecks returns all the available checks that can be included in a benchmark
func (s *BenchmarkService) GetChecks(ctx context.Context, _ *empty.Empty) (*v1.GetChecksResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetBenchmarks returns all the benchmarks as specified by the requests parameters
func (s *BenchmarkService) GetBenchmarks(ctx context.Context, request *v1.GetBenchmarksRequest) (*v1.GetBenchmarksResponse, error) {
	benchmarks, err := s.storage.GetBenchmarks(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarksResponse{Benchmarks: benchmarks}, nil
}

// PostBenchmark creates a new benchmark
func (s *BenchmarkService) PostBenchmark(ctx context.Context, request *v1.Benchmark) (*v1.Benchmark, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new benchmark")
	}
	request.Editable = true // all user generated benchmarks are editable
	id, err := s.storage.AddBenchmark(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	return request, nil
}

// PutBenchmark updates a benchmark
func (s *BenchmarkService) PutBenchmark(ctx context.Context, request *v1.Benchmark) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be specified when updating a benchmark")
	}
	if err := s.storage.UpdateBenchmark(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeleteBenchmark removes a benchmark
func (s *BenchmarkService) DeleteBenchmark(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if err := s.storage.RemoveBenchmark(request.GetId()); err != nil {
		return nil, returnErrorCode(err)
	}
	return &empty.Empty{}, nil
}
