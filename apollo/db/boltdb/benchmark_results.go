package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkResultBucket = "benchmark_results"

func (b *BoltDB) getBenchmarkResult(id string, bucket *bolt.Bucket) (result *v1.BenchmarkResult, exists bool, err error) {
	result = new(v1.BenchmarkResult)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, result)
	return
}

// GetBenchmarkResult returns benchmark with given id.
func (b *BoltDB) GetBenchmarkResult(id string) (benchmark *v1.BenchmarkResult, exists bool, err error) {
	benchmark = new(v1.BenchmarkResult)
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkResultBucket))
		benchmark, exists, err = b.getBenchmarkResult(id, bucket)
		return err
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
		bucket := tx.Bucket([]byte(benchmarkResultBucket))
		_, exists, err := b.getBenchmarkResult(benchmark.Id, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Benchmark result %v cannot be added because it already exists", benchmark.Id)
		}
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(benchmark.Id), bytes)
	})
}
