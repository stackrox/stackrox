package store

import (
	bolt "github.com/etcd-io/bbolt"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/bolthelper"
)

const benchmarkScheduleBucket = "benchmark_schedules"

// Store provides storage functionality for alerts.
type Store interface {
	GetBenchmarkSchedule(name string) (*v1.BenchmarkSchedule, bool, error)
	GetBenchmarkSchedules(request *v1.GetBenchmarkSchedulesRequest) ([]*v1.BenchmarkSchedule, error)
	AddBenchmarkSchedule(schedule *v1.BenchmarkSchedule) (string, error)
	UpdateBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error
	RemoveBenchmarkSchedule(name string) error
}

// New returns a new Store instance using the provided bolt DB instance.
func New(db *bolt.DB) Store {
	bolthelper.RegisterBucketOrPanic(db, benchmarkScheduleBucket)
	return &storeImpl{
		DB: db,
	}
}
