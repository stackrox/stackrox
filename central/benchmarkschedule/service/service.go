package service

import (
	"context"

	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkschedule/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
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
	GetBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*storage.BenchmarkSchedule, error)
	PostBenchmarkSchedule(ctx context.Context, request *storage.BenchmarkSchedule) (*storage.BenchmarkSchedule, error)

	PutBenchmarkSchedule(ctx context.Context, request *storage.BenchmarkSchedule) (*v1.Empty, error)
	GetBenchmarkSchedules(ctx context.Context, request *v1.GetBenchmarkSchedulesRequest) (*v1.GetBenchmarkSchedulesResponse, error)
	DeleteBenchmarkSchedule(ctx context.Context, request *v1.ResourceByID) (*v1.Empty, error)
}

// New returns a new Service instance using the given DataStore.
func New(storage store.Store, datastore benchmarkDataStore.DataStore) Service {
	return &serviceImpl{
		storage:   storage,
		datastore: datastore,
	}
}
