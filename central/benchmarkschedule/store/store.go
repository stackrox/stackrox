package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
)

var benchmarkScheduleBucket = []byte("benchmark_schedules")

// Store provides storage functionality for alerts.
type Store interface {
	GetBenchmarkSchedule(name string) (*storage.BenchmarkSchedule, bool, error)
	GetBenchmarkSchedules(request *v1.GetBenchmarkSchedulesRequest) ([]*storage.BenchmarkSchedule, error)
	AddBenchmarkSchedule(schedule *storage.BenchmarkSchedule) (string, error)
	UpdateBenchmarkSchedule(schedule *storage.BenchmarkSchedule) error
	RemoveBenchmarkSchedule(name string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, benchmarkScheduleBucket)
	return &storeImpl{
		DB: db,
	}
}
