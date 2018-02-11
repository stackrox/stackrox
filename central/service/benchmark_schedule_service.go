package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	"github.com/golang/protobuf/ptypes"
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

// AuthFuncOverride specifies the auth criteria for this API.
func (s *BenchmarkScheduleService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, returnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// GetBenchmarkSchedule returns the current benchmark schedules
func (s *BenchmarkScheduleService) GetBenchmarkSchedule(ctx context.Context, request *v1.GetBenchmarkScheduleRequest) (*v1.BenchmarkSchedule, error) {
	if request.GetName() == "" {
		return nil, status.Error(codes.InvalidArgument, "Name field must be specified when retrieving a benchmark schedule")
	}
	schedule, exists, err := s.storage.GetBenchmarkSchedule(request.GetName())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Schedule with name %v was not found", request.GetName()))
	}
	return schedule, nil
}

// PostBenchmarkSchedule adds a new schedule
func (s *BenchmarkScheduleService) PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	if _, err := benchmarks.ParseHour(request.GetHour()); err != nil {
		return nil, fmt.Errorf("Could not parse hour '%v'", request.GetHour())
	}
	if !benchmarks.ValidDay(request.GetDay()) {
		return nil, fmt.Errorf("'%v' is not a valid day of the week", request.GetDay())
	}
	// TODO(cg) Validate benchmark schedule
	request.LastUpdated = ptypes.TimestampNow()
	if err := s.storage.AddBenchmarkSchedule(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// PutBenchmarkSchedule updates a current schedule
func (s *BenchmarkScheduleService) PutBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	if _, err := benchmarks.ParseHour(request.GetHour()); err != nil {
		return nil, fmt.Errorf("Could not parse hour '%v'", request.GetHour())
	}
	if !benchmarks.ValidDay(request.GetDay()) {
		return nil, fmt.Errorf("'%v' is not a valid day of the week", request.GetDay())
	}
	// TODO(cg) Validate benchmark schedule
	request.LastUpdated = ptypes.TimestampNow()
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
		return nil, returnErrorCode(err)
	}
	return &empty.Empty{}, nil
}
