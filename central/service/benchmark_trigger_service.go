package service

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/datastore"
	"bitbucket.org/stack-rox/apollo/central/db"
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

// NewBenchmarkTriggerService returns the BenchmarkService API.
func NewBenchmarkTriggerService(storage datastore.BenchmarkDataStore, triggerStorage db.BenchmarkTriggerStorage) *BenchmarkTriggerService {
	return &BenchmarkTriggerService{
		storage:        storage,
		triggerStorage: triggerStorage,
	}
}

// BenchmarkTriggerService is the struct that manages the benchmark API
type BenchmarkTriggerService struct {
	storage        datastore.BenchmarkDataStore
	triggerStorage db.BenchmarkTriggerStorage
}

// RegisterServiceServer registers this service with the given gRPC Server.
func (s *BenchmarkTriggerService) RegisterServiceServer(grpcServer *grpc.Server) {
	v1.RegisterBenchmarkTriggerServiceServer(grpcServer, s)
}

// RegisterServiceHandlerFromEndpoint registers this service with the given gRPC Gateway endpoint.
func (s *BenchmarkTriggerService) RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return v1.RegisterBenchmarkTriggerServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
}

// AuthFuncOverride specifies the auth criteria for this API.
func (s *BenchmarkTriggerService) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	return ctx, ReturnErrorCode(or.SensorOrUser().Authorized(ctx))
}

// Trigger triggers a benchmark launch asynchronously.
func (s *BenchmarkTriggerService) Trigger(ctx context.Context, request *v1.BenchmarkTrigger) (*empty.Empty, error) {
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
func (s *BenchmarkTriggerService) GetTriggers(ctx context.Context, request *v1.GetBenchmarkTriggersRequest) (*v1.GetBenchmarkTriggersResponse, error) {
	triggers, err := s.triggerStorage.GetBenchmarkTriggers(request)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.GetBenchmarkTriggersResponse{Triggers: triggers}, nil
}
