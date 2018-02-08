package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkScheduleBucket = "benchmark_schedules"

func (b *BoltDB) getBenchmarkSchedule(name string, bucket *bolt.Bucket) (schedule *v1.BenchmarkSchedule, exists bool, err error) {
	schedule = new(v1.BenchmarkSchedule)
	val := bucket.Get([]byte(name))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, schedule)
	return
}

// GetBenchmarkSchedule returns a benchmark schedule with given id.
func (b *BoltDB) GetBenchmarkSchedule(name string) (schedule *v1.BenchmarkSchedule, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkScheduleBucket))
		schedule, exists, err = b.getBenchmarkSchedule(name, bucket)
		return err
	})
	return
}

// GetBenchmarkSchedules retrieves benchmark schedules matching the request from bolt
func (b *BoltDB) GetBenchmarkSchedules(request *v1.GetBenchmarkSchedulesRequest) ([]*v1.BenchmarkSchedule, error) {
	var schedules []*v1.BenchmarkSchedule
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkScheduleBucket))
		err := b.ForEach(func(k, v []byte) error {
			var schedule v1.BenchmarkSchedule
			if err := proto.Unmarshal(v, &schedule); err != nil {
				return err
			}
			schedules = append(schedules, &schedule)
			return nil
		})
		return err
	})
	return schedules, err
}

// AddBenchmarkSchedule adds a benchmark schedule to bolt
func (b *BoltDB) AddBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(benchmarkScheduleBucket))
		_, exists, err := b.getBenchmarkSchedule(schedule.GetName(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Benchmark Schedule %v cannot be added because it already exists", schedule.GetName())
		}
		bytes, err := proto.Marshal(schedule)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(schedule.GetName()), bytes)
		return err
	})
}

// UpdateBenchmarkSchedule updates a benchmark schedule to bolt
func (b *BoltDB) UpdateBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkScheduleBucket))
		bytes, err := proto.Marshal(schedule)
		if err != nil {
			return err
		}
		err = b.Put([]byte(schedule.GetName()), bytes)
		return err
	})
}

// RemoveBenchmarkSchedule removes a benchmark schedule
func (b *BoltDB) RemoveBenchmarkSchedule(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkScheduleBucket))
		key := []byte(name)
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Benchmark Schedule", ID: name}
		}
		return b.Delete(key)
	})
}
