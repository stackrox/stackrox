package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkBucket = "benchmarks"

// GetBenchmark returns benchmark with given id.
func (b *BoltDB) GetBenchmark(id string) (benchmark *v1.BenchmarkPayload, exists bool, err error) {
	benchmark = new(v1.BenchmarkPayload)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		val := b.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, benchmark)
	})

	return
}

// GetBenchmarks retrieves benchmarks matching the request from bolt
func (b *BoltDB) GetBenchmarks(*v1.GetBenchmarksRequest) ([]*v1.BenchmarkPayload, error) {
	var benchmarks []*v1.BenchmarkPayload
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		b.ForEach(func(k, v []byte) error {
			var benchmark v1.BenchmarkPayload
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
func (b *BoltDB) AddBenchmark(benchmark *v1.BenchmarkPayload) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		err = b.Put([]byte(benchmark.Id), bytes)
		return err
	})
}
