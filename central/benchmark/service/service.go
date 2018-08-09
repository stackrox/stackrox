package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
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

	GetBenchmark(ctx context.Context, request *v1.ResourceByID) (*v1.Benchmark, error)
	GetChecks(ctx context.Context, _ *empty.Empty) (*v1.GetChecksResponse, error)
	GetBenchmarks(ctx context.Context, request *v1.GetBenchmarksRequest) (*v1.GetBenchmarksResponse, error)
	PostBenchmark(ctx context.Context, request *v1.Benchmark) (*v1.Benchmark, error)
	PutBenchmark(ctx context.Context, request *v1.Benchmark) (*empty.Empty, error)
	DeleteBenchmark(ctx context.Context, request *v1.ResourceByID) (*empty.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}
