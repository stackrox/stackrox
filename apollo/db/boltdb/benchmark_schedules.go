package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkScheduleBucket = "benchmark_schedules"

// GetBenchmarkSchedule returns a benchmark schedule with given id.
func (b *BoltDB) GetBenchmarkSchedule(name string) (schedule *v1.BenchmarkSchedule, exists bool, err error) {
	schedule = new(v1.BenchmarkSchedule)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkScheduleBucket))
		val := b.Get([]byte(name))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, schedule)
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

func (b *BoltDB) upsertBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
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

// AddBenchmarkSchedule adds a benchmark schedule to bolt
func (b *BoltDB) AddBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
	return b.upsertBenchmarkSchedule(schedule)
}

// UpdateBenchmarkSchedule updates a benchmark schedule to bolt
func (b *BoltDB) UpdateBenchmarkSchedule(schedule *v1.BenchmarkSchedule) error {
	return b.upsertBenchmarkSchedule(schedule)
}

// RemoveBenchmarkSchedule removes a benchmark schedule
func (b *BoltDB) RemoveBenchmarkSchedule(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkScheduleBucket))
		return b.Delete([]byte(name))
	})
}
