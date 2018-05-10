package boltdb

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkBucket = "benchmarks"

func (b *BoltDB) getBenchmark(id string, bucket *bolt.Bucket) (benchmark *v1.Benchmark, exists bool, err error) {
	benchmark = new(v1.Benchmark)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, benchmark)
	return
}

// GetBenchmark returns benchmark with given id.
func (b *BoltDB) GetBenchmark(id string) (benchmark *v1.Benchmark, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "Benchmark")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		benchmark, exists, err = b.getBenchmark(id, bucket)
		return err
	})
	return
}

// GetBenchmarks retrieves benchmarks matching the request from bolt
func (b *BoltDB) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*v1.Benchmark, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "Benchmark")
	var benchmarks []*v1.Benchmark
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkBucket))
		return b.ForEach(func(k, v []byte) error {
			var benchmark v1.Benchmark
			if err := proto.Unmarshal(v, &benchmark); err != nil {
				return err
			}
			benchmarks = append(benchmarks, &benchmark)
			return nil
		})
	})
	return benchmarks, err
}

// AddBenchmark adds a benchmark to bolt
func (b *BoltDB) AddBenchmark(benchmark *v1.Benchmark) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "Benchmark")
	benchmark.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		_, exists, err := b.getBenchmark(benchmark.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Benchmark %v cannot be added because it already exists", benchmark.GetId())
		}
		if err := checkUniqueKeyExistsAndInsert(tx, benchmarkBucket, benchmark.GetId(), benchmark.GetName()); err != nil {
			return fmt.Errorf("Could not add benchmark due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(benchmark.GetId()), bytes)
	})
	return benchmark.GetId(), err
}

// UpdateBenchmark updates a benchmark to bolt
func (b *BoltDB) UpdateBenchmark(benchmark *v1.Benchmark) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Update", "Benchmark")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		currBenchmark, exists, err := b.getBenchmark(benchmark.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists && !currBenchmark.Editable {
			return fmt.Errorf("Cannot update benchmark %v because it cannot be edited", benchmark.GetId())
		}
		// If the update is changing the name, check if the name has already been taken
		if getCurrentUniqueKey(tx, benchmarkBucket, benchmark.GetId()) != benchmark.GetName() {
			if err := checkUniqueKeyExistsAndInsert(tx, benchmarkBucket, benchmark.GetId(), benchmark.GetName()); err != nil {
				return fmt.Errorf("Could not update benchmark due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(benchmark)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(benchmark.GetId()), bytes)
	})
}

// RemoveBenchmark removes a benchmark.
func (b *BoltDB) RemoveBenchmark(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "Benchmark")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkBucket))
		benchmark, exists, err := b.getBenchmark(id, bucket)
		if err != nil {
			return err
		}
		if !exists {
			return db.ErrNotFound{Type: "Benchmark", ID: id}
		}
		if exists && !benchmark.Editable {
			return fmt.Errorf("Cannot remove benchmark %v because it cannot be edited", benchmark.GetId())
		}
		if err := removeUniqueKey(tx, benchmarkBucket, benchmark.GetId()); err != nil {
			return err
		}
		return bucket.Delete([]byte(id))
	})
}
