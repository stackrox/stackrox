package service

import (
	"fmt"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	bDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	btDataStore "github.com/stackrox/rox/central/benchmarktrigger/datastore"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/central/service"
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
	authorizer = or.SensorOrAuthorizer(perrpc.FromMap(map[authz.Authorizer][]string{
		user.With(permissions.View(resources.BenchmarkTrigger)): {
			"/v1.BenchmarkTriggerService/GetTriggers",
		},
		user.With(permissions.Modify(resources.BenchmarkTrigger)): {
			"/v1.BenchmarkTriggerService/Trigger",
		},
	}))
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
	return ctx, service.ReturnErrorCode(authorizer.Authorized(ctx, fullMethodName))
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
