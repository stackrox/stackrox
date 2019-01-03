package store

import (
	"fmt"
	"time"

	"github.com/deckarep/golang-set"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/uuid"
)

type storeImpl struct {
	*bolt.DB
}

func (b *storeImpl) getBenchmarkSchedule(id string, bucket *bolt.Bucket) (schedule *storage.BenchmarkSchedule, exists bool, err error) {
	schedule = new(storage.BenchmarkSchedule)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, schedule)
	return
}

// GetBenchmarkSchedule returns a benchmark schedule with given id.
func (b *storeImpl) GetBenchmarkSchedule(id string) (schedule *storage.BenchmarkSchedule, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "BenchmarkSchedule")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(benchmarkScheduleBucket)
		schedule, exists, err = b.getBenchmarkSchedule(id, bucket)
		return err
	})
	return
}

// GetBenchmarkSchedules retrieves benchmark schedules matching the request from bolt
func (b *storeImpl) GetBenchmarkSchedules(request *v1.GetBenchmarkSchedulesRequest) ([]*storage.BenchmarkSchedule, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "BenchmarkSchedule")
	var schedules []*storage.BenchmarkSchedule
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(benchmarkScheduleBucket)
		err := b.ForEach(func(k, v []byte) error {
			var schedule storage.BenchmarkSchedule
			if err := proto.Unmarshal(v, &schedule); err != nil {
				return err
			}
			schedules = append(schedules, &schedule)
			return nil
		})
		return err
	})
	filteredSchedules := schedules[:0]
	requestClusterSet := newStringSet(request.GetClusterIds())
	for _, schedule := range schedules {
		if request.GetBenchmarkId() != "" && schedule.GetBenchmarkId() != request.GetBenchmarkId() {
			continue
		}
		clusterSet := newStringSet(schedule.GetClusterIds())
		if requestClusterSet.Cardinality() != 0 && clusterSet.Cardinality() != 0 && requestClusterSet.Intersect(clusterSet).Cardinality() == 0 {
			continue
		}
		filteredSchedules = append(filteredSchedules, schedule)
	}
	return filteredSchedules, err
}

// AddBenchmarkSchedule adds a benchmark schedule to bolt
func (b *storeImpl) AddBenchmarkSchedule(schedule *storage.BenchmarkSchedule) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "BenchmarkSchedule")
	schedule.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(benchmarkScheduleBucket)
		_, exists, err := b.getBenchmarkSchedule(schedule.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Benchmark Schedule %v cannot be added because it already exists", schedule.GetId())
		}
		bytes, err := proto.Marshal(schedule)
		if err != nil {
			return err
		}
		err = bucket.Put([]byte(schedule.GetId()), bytes)
		return err
	})
	return schedule.Id, err
}

// UpdateBenchmarkSchedule updates a benchmark schedule to bolt
func (b *storeImpl) UpdateBenchmarkSchedule(schedule *storage.BenchmarkSchedule) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "BenchmarkSchedule")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(benchmarkScheduleBucket)
		bytes, err := proto.Marshal(schedule)
		if err != nil {
			return err
		}
		err = b.Put([]byte(schedule.GetId()), bytes)
		return err
	})
}

// RemoveBenchmarkSchedule removes a benchmark schedule
func (b *storeImpl) RemoveBenchmarkSchedule(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "BenchmarkSchedule")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(benchmarkScheduleBucket)
		key := []byte(id)
		if exists := b.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Benchmark Schedule", ID: id}
		}
		return b.Delete(key)
	})
}

func newStringSet(strs []string) mapset.Set {
	set := mapset.NewSet()
	for _, s := range strs {
		set.Add(s)
	}
	return set
}
