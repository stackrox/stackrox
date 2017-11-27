package boltdb

import "bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"

// AddBenchmark adds a benchmark to bolt
func (b *BoltDB) AddBenchmark(benchmark *v1.BenchmarkPayload) {
	panic("implement me")
}

// GetBenchmarks retrieves benchmarks matching the request from bolt
func (b *BoltDB) GetBenchmarks(request *v1.GetBenchmarksRequest) []*v1.BenchmarkPayload {
	panic("implement me")
}
