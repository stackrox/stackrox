package boltdb

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const benchmarkTriggerBucket = "benchmark_triggers"

// GetBenchmarkTriggers retrieves benchmark triggers from bolt
func (b *BoltDB) GetBenchmarkTriggers(request *v1.GetBenchmarkTriggersRequest) ([]*v1.BenchmarkTrigger, error) {
	var triggers []*v1.BenchmarkTrigger
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(benchmarkTriggerBucket))
		b.ForEach(func(k, v []byte) error {
			var trigger v1.BenchmarkTrigger
			if err := proto.Unmarshal(v, &trigger); err != nil {
				return err
			}
			triggers = append(triggers, &trigger)
			return nil
		})
		return nil
	})
	return triggers, err
}

// AddBenchmarkTrigger inserts a benchmark trigger into Bolt
func (b *BoltDB) AddBenchmarkTrigger(trigger *v1.BenchmarkTrigger) error {
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
