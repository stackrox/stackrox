package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
}

// GetBenchmarkTriggers retrieves benchmark triggers from bolt
func (b *storeImpl) GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*storage.BenchmarkTrigger, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "BenchmarkTrigger")
	var triggers []*storage.BenchmarkTrigger
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkTriggerBucket))
		return b.ForEach(func(k, v []byte) error {
			var trigger storage.BenchmarkTrigger
			if err := proto.Unmarshal(v, &trigger); err != nil {
				return err
			}
			triggers = append(triggers, &trigger)
			return nil
		})
	})
	return triggers, err
}

// AddBenchmarkTrigger inserts a benchmark trigger into Bolt
func (b *storeImpl) AddBenchmarkTrigger(trigger *storage.BenchmarkTrigger) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "BenchmarkTrigger")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkTriggerBucket))
		bytes, err := proto.Marshal(trigger)
		if err != nil {
			return err
		}
		err = b.Put([]byte(trigger.Time.String()), bytes)
		return err
	})
}
