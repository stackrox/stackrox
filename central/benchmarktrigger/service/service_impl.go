package service

import (
	"fmt"

	bDataStore "bitbucket.org/stack-rox/apollo/central/benchmark/datastore"
	btDataStore "bitbucket.org/stack-rox/apollo/central/benchmarktrigger/datastore"
	"bitbucket.org/stack-rox/apollo/central/service"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/grpc/authz/or"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BenchmarkTriggerService is the struct that manages the benchmark API
type serviceImpl struct {
	storage        bDataStore.DataStore
	triggerStorage btDataStore.DataStore
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *serviceImpl) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkTriggerServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *serviceImpl) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkTriggerServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *serviceImpl) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, service.ReturnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// Trigger triggers a benchmark launch asynchronously.
func (s *serviceImpl) Trigger(ctx context.Context, request *v1.BenchmarkTrigger) (*empty.Empty, error) {
	_, exists, err := s.storage.GetBenchmark(request.GetId())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if !exists {
		return nil, status.Error(codes.NotFound, fmt.Sprintf("Benchmark with id %v does not exist", request.GetId()))
	}
	request.Time = ptypes.TimestampNow()
	if err := s.triggerStorage.AddBenchmarkTrigger(request); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &empty.Empty{}, nil
}

// GetTriggers triggers returns all of the manual benchmark triggers.
func (s *serviceImpl) GetTriggers(ctx context.Context, request *v1.GetBenchmarkTriggersRequest) (*v1.GetBenchmarkTriggersResponse, error) {
	triggers, err := s.triggerStorage.GetBenchmarkTriggers(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarkTriggersResponse{Triggers: triggers}, nil
}
