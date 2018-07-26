package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
)

const benchmarkTriggerBucket = "benchmark_triggers"

// Store provides storage functionality for alerts.
type Store interface {
	GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error)
	AddBenchmarkTrigger(trigger *v1.BenchmarkTrigger) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, benchmarkTriggerBucket)
	return &storeImpl{
		DB: db,
	}
}
