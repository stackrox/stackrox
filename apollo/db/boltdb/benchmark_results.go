package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkResultBucket = "benchmark_results"

// GetBenchmarkResult returns benchmark with given id.
func (b *BoltDB) GetBenchmarkResult(id string) (benchmark *v1.BenchmarkResult, exists bool, err error) {
	benchmark = new(v1.BenchmarkResult)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkResultBucket))
		val := b.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, benchmark)
	})

	return
}

// GetBenchmarkResults retrieves benchmarks matching the request from bolt
func (b *BoltDB) GetBenchmarkResults(request *v1.GetBenchmarkResultsRequest) ([]*v1.BenchmarkResult, error) {
	var benchmarks []*v1.BenchmarkResult
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkResultBucket))
		b.ForEach(func(k, v []byte) error {
			var benchmark v1.BenchmarkResult
			if err := proto.Unmarshal(v, &benchmark); err != nil {
				return err
			}
			benchmarks = append(benchmarks, &benchmark)
			return nil
		})
		return nil
	})
	return benchmarks, err
}

// AddBenchmarkResult adds a benchmark to bolt
func (b *BoltDB) AddBenchmarkResult(benchmark *v1.BenchmarkResult) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkResultBucket))
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		err = b.Put([]byte(benchmark.Id), bytes)
		return err
	})
}
