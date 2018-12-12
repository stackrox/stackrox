package datastore

import (
	"github.com/stackrox/rox/central/benchmarktrigger/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*storage.BenchmarkTrigger, error)
	AddBenchmarkTrigger(trigger *storage.BenchmarkTrigger) error
}

// New returns an instance of DataStore.
func New(storage store.Store) DataStore {
	return &datastoreImpl{
		storage: storage,
	}
}
