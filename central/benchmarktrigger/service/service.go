package service

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	bDataStore "github.com/stackrox/rox/central/benchmark/datastore"
	btDataStore "github.com/stackrox/rox/central/benchmarktrigger/datastore"
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
