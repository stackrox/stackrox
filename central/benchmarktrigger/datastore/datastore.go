package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/benchmarktrigger/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error)
	AddBenchmarkTrigger(trigger *v1.BenchmarkTrigger) error
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
