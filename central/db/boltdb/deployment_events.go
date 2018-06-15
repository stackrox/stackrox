package boltdb

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"time"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

const deploymentEventBucket = "deploymentEvents"

// GetDeploymentEvent returns deployment event with given id.
func (b *BoltDB) GetDeploymentEvent(id uint64) (event *v1.DeploymentEvent, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "DeploymentEvent")

	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentEventBucket))

		val := bucket.Get(itob(id))
		if val == nil {
			return nil
		}
		exists = true

		event = new(v1.DeploymentEvent)
		return proto.Unmarshal(val, event)
	})
	return
}

// GetDeploymentEventIds returns the list of all ids currently stored in bold.
func (b *BoltDB) GetDeploymentEventIds(clusterID string) ([]uint64, map[string]uint64, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "DeploymentEvent")

	var ids []uint64
	deploymentIDToIds := make(map[string]uint64)
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentEventBucket))
		return bucket.ForEach(func(k, v []byte) error {

			event := new(v1.DeploymentEvent)
			if err := proto.Unmarshal(v, event); err != nil {
				return err
			}

			if event.GetDeployment().GetClusterId() != clusterID {
				return nil
			}

			key := btoi(k)
			depID := event.GetDeployment().GetId()

			ids = append(ids, key)
			deploymentIDToIds[depID] = key
			return nil
		})
	})
	return ids, deploymentIDToIds, err
}

// AddDeploymentEvent adds a deployment event to bolt
func (b *BoltDB) AddDeploymentEvent(event *v1.DeploymentEvent) (uint64, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "DeploymentEvent")

	var id uint64
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentEventBucket))

		var err error
		id, err = bucket.NextSequence()
		if err != nil {
			return err
		}

		val := bucket.Get(itob(id))
		if val != nil {
			return fmt.Errorf("deployment %s cannot be added because it already exists", event.GetDeployment().GetId())
		}

		bytes, err := proto.Marshal(event)
		if err != nil {
			return err
		}

		return bucket.Put(itob(id), bytes)
	})
	return id, err
}

// UpdateDeploymentEvent updates a deployment event to bolt
func (b *BoltDB) UpdateDeploymentEvent(id uint64, event *v1.DeploymentEvent) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Update", "DeploymentEvent")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentEventBucket))

		val := bucket.Get(itob(id))
		if val == nil {
			return fmt.Errorf("deployment %s does not exist in the DB", event.GetDeployment().GetId())
		}

		bytes, err := proto.Marshal(event)
		if err != nil {
			return err
		}
		return bucket.Put(itob(id), bytes)
	})
}

// RemoveDeploymentEvent removes a deployment event from bolt.
func (b *BoltDB) RemoveDeploymentEvent(id uint64) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "DeploymentEvent")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentEventBucket))

		val := bucket.Get(itob(id))
		if val == nil {
			return db.ErrNotFound{Type: "Deployment", ID: strconv.FormatUint(id, 10)}
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
