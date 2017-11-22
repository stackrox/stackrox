package service

import (
	"errors"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/scheduler"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// NewBenchmarkService returns the BenchmarkService API.
func NewBenchmarkService(storage db.Storage, schedule *scheduler.DockerBenchScheduler) *BenchmarkService {
	return &BenchmarkService{
		storage:  storage,
		schedule: schedule,
	}
}

// BenchmarkService is the struct that manages the benchmark API
type BenchmarkService struct {
	storage  db.Storage
	schedule *scheduler.DockerBenchScheduler
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// GetBenchmarkResults retrieves benchmark results based on the request filters
func (s *BenchmarkService) GetBenchmarkResults(ctx context.Context, request *v1.GetBenchmarksRequest) (*v1.GetBenchmarksResponse, error) {
	benchmarks := s.storage.GetBenchmarks(request)
	return &v1.GetBenchmarksResponse{Benchmarks: benchmarks}, nil
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *BenchmarkService) PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkPayload) (*empty.Empty, error) {
	s.storage.AddBenchmark(request)
	return &empty.Empty{}, nil
}

// PostBenchmarkSchedule can trigger a new run, set a schedule, or disable a schedule
func (s *BenchmarkService) PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	if !request.Enable {
		s.schedule.Disable()
		return &empty.Empty{}, nil
	} else if request.IntervalDays == 0 {
		return nil, errors.New("enabling benchmark schedule requires a nonzero interval")
	}

	log.Infof("Enabling benchmark schedule with interval %v days", request.IntervalDays)
	s.schedule.Enable(time.Duration(request.IntervalDays) * time.Hour * 24)
	return &empty.Empty{}, nil
}

// GetBenchmarkSchedule returns the current benchmark schedule
func (s *BenchmarkService) GetBenchmarkSchedule(ctx context.Context, _ *empty.Empty) (*v1.GetBenchmarkScheduleResponse, error) {
	protoNextScheduled, err := ptypes.TimestampProto(s.schedule.NextScheduled)
	if err != nil {
		return nil, err
	}

	return &v1.GetBenchmarkScheduleResponse{
		Enabled:       s.schedule.Enabled,
		IntervalDays:  int64(s.schedule.Interval.Hours() / 24),
		NextScheduled: protoNextScheduled,
	}, nil
}

// TriggerBenchmark triggers a benchmark launch asynchronously
func (s *BenchmarkService) TriggerBenchmark(ctx context.Context, request *v1.TriggerBenchmarkRequest) (*empty.Empty, error) {
	err := s.schedule.Launch()
	return &empty.Empty{}, err
}
