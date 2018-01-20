package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkBucket = "benchmarks"

func (b *BoltDB) getBenchmark(name string, bucket *bolt.Bucket) (benchmark *v1.Benchmark, exists bool, err error) {
	benchmark = new(v1.Benchmark)
	val := bucket.Get([]byte(name))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, benchmark)
	return
}

// GetBenchmark returns benchmark with given id.
func (b *BoltDB) GetBenchmark(name string) (benchmark *v1.Benchmark, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		benchmark, exists, err = b.getBenchmark(name, bucket)
		return err
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

// AddBenchmark adds a benchmark to bolt
func (b *BoltDB) AddBenchmark(benchmark *v1.Benchmark) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		_, exists, err := b.getBenchmark(benchmark.Name, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Benchmark %v cannot be added because it already exists", benchmark.GetName())
		}
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(benchmark.Name), bytes)
	})
}

// UpdateBenchmark updates a benchmark to bolt
func (b *BoltDB) UpdateBenchmark(benchmark *v1.Benchmark) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		currBenchmark, exists, err := b.getBenchmark(benchmark.Name, bucket)
		if err != nil {
			return err
		}
		if exists && !currBenchmark.Editable {
			return fmt.Errorf("Cannot update benchmark %v because it cannot be edited", benchmark.Name)
		}
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(benchmark.Name), bytes)
	})
}

// RemoveBenchmark removes a benchmark.
func (b *BoltDB) RemoveBenchmark(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		benchmark, exists, err := b.getBenchmark(name, bucket)
		if err != nil {
			return err
		}
		if exists && !benchmark.Editable {
			return fmt.Errorf("Cannot remove benchmark %v because it cannot be edited", benchmark.Name)
		}
		return bucket.Delete([]byte(name))
	})
}
