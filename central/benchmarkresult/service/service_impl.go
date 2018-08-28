package service

import (
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/hashicorp/golang-lru"
	benchmarkscanStore "github.com/stackrox/rox/central/benchmarkscan/store"
	benchmarkscheduleStore "github.com/stackrox/rox/central/benchmarkschedule/store"
	notifierProcessor "github.com/stackrox/rox/central/notifier/processor"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type serviceImpl struct {
	resultStore            benchmarkscanStore.Store
	scheduleStore          benchmarkscheduleStore.Store
	notificationsProcessor notifierProcessor.Processor
	cache                  *lru.Cache
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkResultsServiceServer(grpcServer, s)
}

// RegisterServiceHandler registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandler(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	return v1.RegisterBenchmarkResultsServiceHandler(ctx, mux, conn)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, idcheck.SensorsOnly().Authorized(ctx, fullMethodName)
}

// PostBenchmarkResult inserts a new benchmark result into the system
func (s *serviceImpl) PostBenchmarkResult(ctx context.Context, request *v1.BenchmarkResult) (*v1.Empty, error) {
	if err := s.resultStore.AddBenchmarkResult(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if request.GetReason() == v1.BenchmarkReason_SCHEDULED {
		if _, ok := s.cache.Get(request.GetScanId()); ok {
			// This means that the scan id has already been processed and an alert about benchmarks coming in was already sent
			return &v1.Empty{}, nil
		}
		s.cache.Add(request.GetScanId(), struct{}{})
		schedule, exists, err := s.scheduleStore.GetBenchmarkSchedule(request.GetId())
		if err != nil {
			log.Errorf("Error retrieving benchmark schedule %v: %+v", request.GetId(), err)
			return &v1.Empty{}, nil
		} else if !exists {
			log.Errorf("Benchmark schedule %v does not exist", request.GetId())
			return &v1.Empty{}, nil
		}
		s.notificationsProcessor.ProcessBenchmark(schedule)
	}
	return &v1.Empty{}, nil
}
