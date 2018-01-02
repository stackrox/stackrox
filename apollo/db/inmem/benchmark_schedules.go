package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
)

type benchmarkScheduleStore struct {
	db.BenchmarkScheduleStorage
}

func newBenchmarkScheduleStore(persistent db.BenchmarkScheduleStorage) *benchmarkScheduleStore {
	return &benchmarkScheduleStore{
		BenchmarkScheduleStorage: persistent,
	}
}
