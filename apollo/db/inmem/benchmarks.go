package inmem

import (
	"bitbucket.org/stack-rox/apollo/apollo/db"
)

type benchmarkStore struct {
	db.BenchmarkStorage
}

func newBenchmarkStore(persistent db.BenchmarkStorage) *benchmarkStore {
	return &benchmarkStore{
		BenchmarkStorage: persistent,
	}
}
