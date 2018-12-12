package datastore

import (
	"github.com/stackrox/rox/central/benchmark/store"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// DataStore is the entry point for modifying Cluster data.
type DataStore interface {
	GetBenchmark(id string) (*storage.Benchmark, bool, error)
	GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*storage.Benchmark, error)
	AddBenchmark(benchmark *storage.Benchmark) (string, error)
	UpdateBenchmark(benchmark *storage.Benchmark) error
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
