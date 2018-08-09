package store

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/globaldb/ops"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dberrors"
)

type storeImpl struct {
	*bolt.DB
}

// GetSensorEvent returns sensor event with given id.
func (b *storeImpl) GetSensorEvent(id uint64) (event *v1.SensorEvent, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "SensorEvent")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sensorEventBucket))

		val := bucket.Get(itob(id))
		if val == nil {
			return nil
		}
		exists = true

		event = new(v1.SensorEvent)
		return proto.Unmarshal(val, event)
	})
	return
}

// GetSensorEventIds returns the list of all ids currently stored in bold.
func (b *storeImpl) GetSensorEventIds(clusterID string) ([]uint64, map[string]uint64, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "SensorEvent")

	var ids []uint64
	sensorIDToIds := make(map[string]uint64)
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sensorEventBucket))
		return bucket.ForEach(func(k, v []byte) error {

			event := new(v1.SensorEvent)
			if err := proto.Unmarshal(v, event); err != nil {
				return err
			}

			if event.GetClusterId() != clusterID {
				return nil
			}

			key := btoi(k)
			depID := event.GetId()

			ids = append(ids, key)
			sensorIDToIds[depID] = key
			return nil
		})
	})
	return ids, sensorIDToIds, err
}

// AddSensorEvent adds a sensor event to bolt
func (b *storeImpl) AddSensorEvent(event *v1.SensorEvent) (uint64, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "SensorEvent")

	var id uint64
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sensorEventBucket))

		var err error
		id, err = bucket.NextSequence()
		if err != nil {
			return err
		}

		val := bucket.Get(itob(id))
		if val != nil {
			return fmt.Errorf("sensor event %s cannot be added because it already exists", event.GetId())
		}

		bytes, err := proto.Marshal(event)
		if err != nil {
			return err
		}

		return bucket.Put(itob(id), bytes)
	})
	return id, err
}

// UpdateSensorEvent updates a sensor event to bolt
func (b *storeImpl) UpdateSensorEvent(id uint64, event *v1.SensorEvent) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "SensorEvent")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sensorEventBucket))

		val := bucket.Get(itob(id))
		if val == nil {
			return fmt.Errorf("sensor event %s does not exist in the DB", event.GetId())
		}

		bytes, err := proto.Marshal(event)
		if err != nil {
			return err
		}
		return bucket.Put(itob(id), bytes)
	})
}

// RemoveSensorEvent removes a sensor event from bolt.
func (b *storeImpl) RemoveSensorEvent(id uint64) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "SensorEvent")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(sensorEventBucket))

		val := bucket.Get(itob(id))
		if val == nil {
			return dberrors.ErrNotFound{Type: "SensorEvent", ID: strconv.FormatUint(id, 10)}
		}

		return bucket.Delete(itob(id))
	})
}

func itob(v uint64) []byte {
	b := make([]byte, 8, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}
