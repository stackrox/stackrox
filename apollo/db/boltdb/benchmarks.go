package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkBucket = "benchmarks"

// GetBenchmark returns benchmark with given id.
func (b *BoltDB) GetBenchmark(name string) (benchmark *v1.Benchmark, exists bool, err error) {
	benchmark = new(v1.Benchmark)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		val := b.Get([]byte(name))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, benchmark)
	})

	return
}

// GetBenchmarks retrieves benchmarks matching the request from bolt
func (b *BoltDB) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error) {
	var benchmarks []*v1.Benchmark
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		b.ForEach(func(k, v []byte) error {
			var benchmark v1.Benchmark
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

func (b *BoltDB) upsertBenchmark(benchmark *v1.Benchmark) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		err = b.Put([]byte(benchmark.Name), bytes)
		return err
	})
}

// AddBenchmark adds a benchmark to bolt
func (b *BoltDB) AddBenchmark(benchmark *v1.Benchmark) error {
	return b.upsertBenchmark(benchmark)
}

// UpdateBenchmark updates a benchmark to bolt
func (b *BoltDB) UpdateBenchmark(benchmark *v1.Benchmark) error {
	return b.upsertBenchmark(benchmark)
}

// RemoveBenchmark removes a benchmark.
func (b *BoltDB) RemoveBenchmark(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		return b.Delete([]byte(name))
	})
}
