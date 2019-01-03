package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/uuid"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getBenchmark(id string, bucket *bolt.Bucket) (benchmark *storage.Benchmark, exists bool, err error) {
	benchmark = new(storage.Benchmark)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, benchmark)
	return
}

// GetBenchmark returns benchmark with given id.
func (b *storeImpl) GetBenchmark(id string) (benchmark *storage.Benchmark, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Benchmark")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(benchmarkBucket)
		benchmark, exists, err = b.getBenchmark(id, bucket)
		return err
	})
	return
}

// GetBenchmarks retrieves benchmarks matching the request from bolt
func (b *storeImpl) GetBenchmarks(request *v1.GetBenchmarksRequest) ([]*storage.Benchmark, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Benchmark")
	var benchmarks []*storage.Benchmark
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(benchmarkBucket)
		return b.ForEach(func(k, v []byte) error {
			var benchmark storage.Benchmark
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
func (b *storeImpl) AddBenchmark(benchmark *storage.Benchmark) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Benchmark")
	benchmark.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(benchmarkBucket)
		_, exists, err := b.getBenchmark(benchmark.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Benchmark %v cannot be added because it already exists", benchmark.GetId())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, benchmarkBucket, benchmark.GetId(), benchmark.GetName()); err != nil {
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
func (b *storeImpl) UpdateBenchmark(benchmark *storage.Benchmark) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Benchmark")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(benchmarkBucket)
		currBenchmark, exists, err := b.getBenchmark(benchmark.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists && !currBenchmark.Editable {
			return fmt.Errorf("Cannot update benchmark %v because it cannot be edited", benchmark.GetId())
		}
		// If the update is changing the name, check if the name has already been taken
		if val, _ := secondarykey.GetCurrentUniqueKey(tx, benchmarkBucket, benchmark.GetId()); val != benchmark.GetName() {
			if err := secondarykey.UpdateUniqueKey(tx, benchmarkBucket, benchmark.GetId(), benchmark.GetName()); err != nil {
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
func (b *storeImpl) RemoveBenchmark(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Benchmark")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(benchmarkBucket)
		benchmark, exists, err := b.getBenchmark(id, bucket)
		if err != nil {
			return err
		}
		if !exists {
			return dberrors.ErrNotFound{Type: "Benchmark", ID: id}
		}
		if exists && !benchmark.Editable {
			return fmt.Errorf("Cannot remove benchmark %v because it cannot be edited", benchmark.GetId())
		}
		if err := secondarykey.RemoveUniqueKey(tx, benchmarkBucket, benchmark.GetId()); err != nil {
			return err
		}
		return bucket.Delete([]byte(id))
	})
}
