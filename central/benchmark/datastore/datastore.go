package datastore

import (
	"bitbucket.org/stack-rox/apollo/central/benchmark/store"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetBenchmark(id string) (*v1.Benchmark, bool, error)
	GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error)
	AddBenchmark(benchmark *v1.Benchmark) (string, error)
	UpdateBenchmark(benchmark *v1.Benchmark) error
	RemoveBenchmark(id string) error
}

// New returns an instance of DataStore.
func New(storage store.Store) (DataStore, error) {
	ds := &datastoreImpl{
		storage: storage,
	}
	if err := ds.loadDefaults(); err != nil {
		return nil, err
	}
	return ds, nil
}
