package store

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/secondarykey"
)

type storeImpl struct {
	*bolt.DB
}

// GetProcessIndicator returns indicator with given id.
func (b *storeImpl) GetProcessIndicator(id string) (indicator *v1.ProcessIndicator, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ProcessIndicator")
	indicator = new(v1.ProcessIndicator)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(processIndicatorBucket))
		val := b.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, indicator)
	})
	return
}

func (b *storeImpl) GetProcessIndicators() ([]*v1.ProcessIndicator, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ProcessIndicator")
	var indicators []*v1.ProcessIndicator
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(processIndicatorBucket))
		return b.ForEach(func(k, v []byte) error {
			var indicator v1.ProcessIndicator
			if err := proto.Unmarshal(v, &indicator); err != nil {
				return err
			}
			indicators = append(indicators, &indicator)
			return nil
		})
	})
	return indicators, err
}

// get the value of the secondary key
func getSecondaryKey(indicator *v1.ProcessIndicator) string {
	signal := indicator.GetSignal()
	processSignal := signal.GetProcessSignal()
	return fmt.Sprintf("%s %s %s %s", signal.GetContainerId(), processSignal.GetExecFilePath(),
		processSignal.GetName(), processSignal.GetCommandLine())
}

func (b *storeImpl) AddProcessIndicator(indicator *v1.ProcessIndicator) (inserted bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "ProcessIndicator")
	err = b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(processIndicatorBucket))
		bytes, err := proto.Marshal(indicator)
		if err != nil {
			return err
		}
		uniqueField := getSecondaryKey(indicator)
		if _, exists := secondarykey.GetCurrentUniqueKey(tx, processIndicatorBucket, uniqueField); exists {
			return nil
		}
		inserted = true
		if err := secondarykey.InsertUniqueKey(tx, processIndicatorBucket, uniqueField, uniqueField); err != nil {
			return err
		}
		return bucket.Put([]byte(indicator.GetId()), bytes)
	})
	return
}

func (b *storeImpl) RemoveProcessIndicator(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "ProcessIndicator")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(processIndicatorBucket))
		return bucket.Delete([]byte(id))
	})
}
