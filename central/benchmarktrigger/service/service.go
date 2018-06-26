package service

import (
	"context"

	bDataStore "bitbucket.org/stack-rox/apollo/central/benchmark/datastore"
	btDataStore "bitbucket.org/stack-rox/apollo/central/benchmarktrigger/datastore"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	RegisterServiceServer(grpcServer *grpc.Server)
	RegisterServiceHandlerFromEndpoint(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	Trigger(ctx context.Context, request *v1.BenchmarkTrigger) (*empty.Empty, error)
	GetTriggers(ctx context.Context, request *v1.GetBenchmarkTriggersRequest) (*v1.GetBenchmarkTriggersResponse, error)
}

// New returns a new Service instance using the given DataStore.
func New(triggerStorage btDataStore.DataStore, storage bDataStore.DataStore) Service {
	return &serviceImpl{
		storage:        storage,
		triggerStorage: triggerStorage,
	}
}
