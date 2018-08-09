package datastore

import (
	"github.com/stackrox/rox/central/benchmark/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/logging"
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
