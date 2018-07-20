package store

import (
	"time"

	"bitbucket.org/stack-rox/apollo/central/globaldb/ops"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

type storeImpl struct {
	*bolt.DB
}

// GetBenchmarkTriggers retrieves benchmark triggers from bolt
func (b *storeImpl) GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "BenchmarkTrigger")
	var triggers []*v1.BenchmarkTrigger
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkTriggerBucket))
		return b.ForEach(func(k, v []byte) error {
			var trigger v1.BenchmarkTrigger
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
func (b *storeImpl) AddBenchmarkTrigger(trigger *v1.BenchmarkTrigger) error {
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
