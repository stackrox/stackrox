package service

import (
	"context"

	"github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/grpc"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// Service provides the interface to the microservice that serves alert data.
type Service interface {
	grpc.APIService

	AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)

	GetBenchmark(ctx context.Context, request *v1.ResourceByID) (*v1.Benchmark, error)
	GetChecks(ctx context.Context, _ *v1.Empty) (*v1.GetChecksResponse, error)
	GetBenchmarks(ctx context.Context, request *v1.GetBenchmarksRequest) (*v1.GetBenchmarksResponse, error)
	PostBenchmark(ctx context.Context, request *v1.Benchmark) (*v1.Benchmark, error)
	PutBenchmark(ctx context.Context, request *v1.Benchmark) (*v1.Empty, error)
	DeleteBenchmark(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(datastore datastore.DataStore) Service {
	return &serviceImpl{
		datastore: datastore,
	}
}
