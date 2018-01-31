package inmem

import (
	"bitbucket.org/stack-rox/apollo/central/db"
)

type benchmarkResultStore struct {
	db.BenchmarkScansStorage
}

func newBenchmarkResultsStore(persistent db.BenchmarkScansStorage) *benchmarkResultStore {
	return &benchmarkResultStore{
		BenchmarkScansStorage: persistent,
	}
}
