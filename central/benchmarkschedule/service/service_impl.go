package service

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkschedule/store"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/service"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/benchmarks"
	"github.com/stackrox/rox/pkg/errorhelpers"
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
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.BenchmarkSchedule)): {
			"/v1.BenchmarkScheduleService/GetBenchmarkSchedule",
			"/v1.BenchmarkScheduleService/GetBenchmarkSchedules",
		},
		user.With(permissions.Modify(resources.BenchmarkSchedule)): {
			"/v1.BenchmarkScheduleService/PostBenchmarkSchedule",
			"/v1.BenchmarkScheduleService/PutBenchmarkSchedule",
			"/v1.BenchmarkScheduleService/DeleteBenchmarkSchedule",
		},
	}))
)

// BenchmarkScheduleService is the struct that manages the benchmark API
type serviceImpl struct {
	storage   store.Store
	datastore benchmarkDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkScheduleServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkScheduleServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
}

// GetBenchmarkSchedule returns the current benchmark schedules
func (s *serviceImpl) GetBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*v1.BenchmarkSchedule, error) {
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

func (s *serviceImpl) validateBenchmarkSchedule(request *v1.BenchmarkSchedule) error {
	errorList := errorhelpers.NewErrorList("Validation")
	if request.GetBenchmarkId() == "" {
		errorList.AddString("Benchmark id must be defined ")
	}
	_, exists, err := s.datastore.GetBenchmark(request.GetBenchmarkId())
	if err != nil {
		return err
	}
	if !exists {
		errorList.AddString(fmt.Sprintf("Benchmark with id '%v' does not exist", request.GetBenchmarkId()))
	}
	if request.GetBenchmarkName() == "" {
		errorList.AddString("Benchmark name must be defined")
	}
	if _, err := benchmarks.ParseHour(request.GetHour()); err != nil {
		errorList.AddString(fmt.Sprintf("Could not parse hour '%v'", request.GetHour()))
	}
	if !benchmarks.ValidDay(request.GetDay()) {
		errorList.AddString(fmt.Sprintf("'%v' is not a valid day of the week", request.GetDay()))
	}
	return errorList.ToError()
}

// PostBenchmarkSchedule adds a new schedule
func (s *serviceImpl) PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*v1.BenchmarkSchedule, error) {
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
func (s *serviceImpl) PutBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
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
func (s *serviceImpl) GetBenchmarkSchedules(ctx context.Context, request *v1.GetBenchmarkSchedulesRequest) (*v1.GetBenchmarkSchedulesResponse, error) {
	schedules, err := s.storage.GetBenchmarkSchedules(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarkSchedulesResponse{
		Schedules: schedules,
	}, nil
}

// DeleteBenchmarkSchedule removes a benchmark schedule
func (s *serviceImpl) DeleteBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error) {
	if err := s.storage.RemoveBenchmarkSchedule(request.GetId()); err != nil {
		return nil, service.ReturnErrorCode(err)
	}
	return &empty.Empty{}, nil
}
