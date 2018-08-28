package service

import (
	"fmt"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/perrpc"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	authorizer = perrpc.FromMap(map[authz.Authorizer][]string{
		or.SensorOrAuthorizer(user.With(permissions.View(resources.Benchmark))): {
			"/v1.BenchmarkService/GetChecks",
			"/v1.BenchmarkService/GetBenchmark",
			"/v1.BenchmarkService/GetBenchmarks",
		},
		user.With(permissions.Modify(resources.Benchmark)): {
			"/v1.BenchmarkService/PostBenchmark",
			"/v1.BenchmarkService/PutBenchmark",
			"/v1.BenchmarkService/DeleteBenchmark",
		},
	})
)

// ClusterService is the struct that manages the cluster API
type serviceImpl struct {
	datastore datastore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterBenchmarkServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, authorizer.Authorized(ctx, fullMethodName)
}

// GetBenchmark returns the benchmark by the passed name
func (s *serviceImpl) GetBenchmark(ctx context.Context, request *v1.ResourceByID) (*v1.Benchmark, error) {
	benchmark, exists, err := s.datastore.GetBenchmark(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Benchmark with id %v is not found", request.GetId()))
	}
	return benchmark, nil
}

// GetChecks returns all the available checks that can be included in a benchmark
func (s *serviceImpl) GetChecks(ctx context.Context, _ *v1.Empty) (*v1.GetChecksResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// GetBenchmarks returns all the benchmarks as specified by the requests parameters
func (s *serviceImpl) GetBenchmarks(ctx context.Context, request *v1.GetBenchmarksRequest) (*v1.GetBenchmarksResponse, error) {
	benchmarks, err := s.datastore.GetBenchmarks(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarksResponse{Benchmarks: benchmarks}, nil
}

// PostBenchmark creates a new benchmark
func (s *serviceImpl) PostBenchmark(ctx context.Context, request *v1.Benchmark) (*v1.Benchmark, error) {
	if request.GetId() != "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be empty when posting a new benchmark")
	}
	request.Editable = true // all user generated benchmarks are editable
	id, err := s.datastore.AddBenchmark(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	return request, nil
}

// PutBenchmark updates a benchmark
func (s *serviceImpl) PutBenchmark(ctx context.Context, request *v1.Benchmark) (*v1.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Id field should be specified when updating a benchmark")
	}
	if err := s.datastore.UpdateBenchmark(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.Empty{}, nil
}

// DeleteBenchmark removes a benchmark
func (s *serviceImpl) DeleteBenchmark(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error) {
	if err := s.datastore.RemoveBenchmark(request.GetId()); err != nil {
		return nil, err
	}
	return &v1.Empty{}, nil
}
