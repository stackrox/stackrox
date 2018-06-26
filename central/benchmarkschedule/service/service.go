package service

import (
	"context"

	benchmarkDataStore "bitbucket.org/stack-rox/apollo/central/benchmark/datastore"
	"bitbucket.org/stack-rox/apollo/central/benchmarkschedule/store"
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
	GetBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*v1.BenchmarkSchedule, error)
	PostBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*v1.BenchmarkSchedule, error)

	PutBenchmarkSchedule(ctx context.Context, request *v1.BenchmarkSchedule) (*empty.Empty, error)
	GetBenchmarkSchedules(ctx context.Context, request *v1.GetBenchmarkSchedulesRequest) (*v1.GetBenchmarkSchedulesResponse, error)
	DeleteBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store, datastore benchmarkDataStore.DataStore) Service {
	return &serviceImpl{
		storage:   storage,
		datastore: datastore,
	}
}
