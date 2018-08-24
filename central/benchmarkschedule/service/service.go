package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	benchmarkDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	"github.com/stackrox/rox/central/benchmarkschedule/store"
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
