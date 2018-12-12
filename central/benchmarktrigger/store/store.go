package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const benchmarkTriggerBucket = "benchmark_triggers"

// Store provides storage functionality for alerts.
type Store interface {
	GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*storage.BenchmarkTrigger, error)
	AddBenchmarkTrigger(trigger *storage.BenchmarkTrigger) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, benchmarkTriggerBucket)
	return &storeImpl{
		DB: db,
	}
}
