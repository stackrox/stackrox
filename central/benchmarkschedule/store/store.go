package store

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/bolthelper"
	"github.com/boltdb/bolt"
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
	bolthelper.RegisterBucket(db, benchmarkScheduleBucket)
	return &storeImpl{
		DB: db,
	}
}
