package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const benchmarkBucket = "benchmarks"

// Store provides storage functionality for alerts.
type Store interface {
	GetBenchmark(id string) (*storage.Benchmark, bool, error)
	GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*storage.Benchmark, error)
	AddBenchmark(benchmark *storage.Benchmark) (string, error)
	UpdateBenchmark(benchmark *storage.Benchmark) error
	RemoveBenchmark(id string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, benchmarkBucket)
	return &storeImpl{
		DB: db,
	}
}
