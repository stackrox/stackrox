package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/benchmarks"
	"bitbucket.org/stack-rox/apollo/pkg/errorHelpers"
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
func (s *BenchmarkScheduleService) GetBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*v1.BenchmarkSchedule, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "Name field must be specified when retrieving a benchmark schedule")
	}
	schedule, exists, err := s.storage.GetBenchmarkSchedule(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Schedule with name %v was not found", request.GetId()))
	}
	return schedule, nil
}

func (s *BenchmarkScheduleService) validateBenchmarkSchedule(request *v1.BenchmarkSchedule) error {
	var errs []string
	if request.GetBenchmarkId() == "" {
		errs = append(errs, "Benchmark id must be defined ")
	}
	_, exists, err := s.storage.GetBenchmark(request.GetBenchmarkId())
	if err != nil {
		return err
	}
	if !exists {
		errs = append(errs, fmt.Sprintf("Benchmark with id '%v' does not exist", request.GetBenchmarkId()))
	}
	if request.GetBenchmarkName() == "" {
		errs = append(errs, "Benchmark name must be defined")
	}
	if _, err := benchmarks.ParseHour(request.GetHour()); err != nil {
		errs = append(errs, fmt.Sprintf("Could not parse hour '%v'", request.GetHour()))
	}
	if !benchmarks.ValidDay(request.GetDay()) {
		errs = append(errs, fmt.Sprintf("'%v' is not a valid day of the week", request.GetDay()))
	}
	if len(errs) > 0 {
		return errorhelpers.FormatErrorStrings("Validation", errs)
	}
	return nil
}

// PostBenchmarkSchedule adds a new schedule
func (s *BenchmarkScheduleService) PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*v1.BenchmarkSchedule, error) {
	if err := s.validateBenchmarkSchedule(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	request.LastUpdated = ptypes.TimestampNow()
	id, err := s.storage.AddBenchmarkSchedule(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	request.Id = id
	return request, nil
}

// PutBenchmarkSchedule updates a current schedule
func (s *BenchmarkScheduleService) PutBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	if request.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "id must be defined")
	}
	if err := s.validateBenchmarkSchedule(request); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
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
func (s *BenchmarkScheduleService) DeleteBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if err := s.storage.RemoveBenchmarkSchedule(request.GetId()); err != nil {
		return nil, returnErrorCode(err)
	}
	return &empty.Empty{}, nil
}
