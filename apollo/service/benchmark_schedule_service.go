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

// NewBenchmarkScheduleService returns the BenchmarkService API.
func NewBenchmarkScheduleService(storage db.Storage) *BenchmarkScheduleService {
	return &BenchmarkScheduleService{
		storage: storage,
	}
}

// BenchmarkScheduleService is the struct that manages the benchmark API
type BenchmarkScheduleService struct {
	storage db.Storage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkScheduleService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkScheduleServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkScheduleService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkScheduleServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// PostBenchmarkSchedule adds a new schedule
func (s *BenchmarkScheduleService) PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	// TODO(cg) Validate benchmark schedule
	if err := s.storage.AddBenchmarkSchedule(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// PutBenchmarkSchedule updates a current schedule
func (s *BenchmarkScheduleService) PutBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	// TODO(cg) Validate benchmark schedule
	if err := s.storage.UpdateBenchmarkSchedule(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// GetBenchmarkSchedules returns the current benchmark schedules
func (s *BenchmarkScheduleService) GetBenchmarkSchedules(ctx context.Context, request *v1.GetBenchmarkSchedulesRequest) (*v1.GetBenchmarkSchedulesResponse, error) {
	schedules, err := s.storage.GetBenchmarkSchedules(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarkSchedulesResponse{
		Schedules: schedules,
	}, nil
}

// DeleteBenchmarkSchedule removes a benchmark schedule
func (s *BenchmarkScheduleService) DeleteBenchmarkSchedule(ctx context.Context, request *v1.DeleteBenchmarkScheduleRequest) (*empty.Empty, error) {
	if err := s.storage.RemoveBenchmarkSchedule(request.Name); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}
