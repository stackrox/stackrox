package service

import (
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/db"
	"bitbucket.org/stack-rox/apollo/apollo/scheduler"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	storage  db.BenchmarkStorage
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

// PostBenchmarkSchedule can trigger a new run, set a schedule, or disable a schedule
func (s *BenchmarkService) PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error) {
	if !request.Enable {
		s.schedule.Disable()
		return &empty.Empty{}, nil
	} else if request.IntervalDays == 0 {
		return nil, status.Error(codes.Internal, "enabling benchmark schedule requires a nonzero interval")
	}

	log.Infof("Enabling benchmark schedule with interval %v days", request.IntervalDays)
	s.schedule.Enable(time.Duration(request.IntervalDays) * time.Hour * 24)
	return &empty.Empty{}, nil
}

// GetBenchmarkSchedule returns the current benchmark schedule
func (s *BenchmarkService) GetBenchmarkSchedule(ctx context.Context, _ *empty.Empty) (*v1.GetBenchmarkScheduleResponse, error) {
	return &v1.GetBenchmarkScheduleResponse{
		Enabled:       s.schedule.Enabled,
		IntervalDays:  int64(s.schedule.Interval.Hours() / 24),
		CurrentScanId: s.schedule.CurrentScanID,
	}, nil
}

// TriggerBenchmark triggers a benchmark launch asynchronously
func (s *BenchmarkService) TriggerBenchmark(ctx context.Context, request *v1.TriggerBenchmarkRequest) (*empty.Empty, error) {
	s.schedule.Trigger()
	return &empty.Empty{}, nil
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
func (s *BenchmarkService) PostBenchmark(ctx context.Context, request *v1.Benchmark) (*empty.Empty, error) {
	if err := s.storage.AddBenchmark(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// PutBenchmark updates a benchmark
func (s *BenchmarkService) PutBenchmark(ctx context.Context, request *v1.Benchmark) (*empty.Empty, error) {
	if err := s.storage.UpdateBenchmark(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// DeleteBenchmark removes a benchmark
func (s *BenchmarkService) DeleteBenchmark(ctx context.Context, request *v1.DeleteBenchmarkRequest) (*empty.Empty, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
