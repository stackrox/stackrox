package boltdb

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

const deploymentBucket = "deployments"

func (b *BoltDB) getDeployment(id string, bucket *bolt.Bucket) (deployment *v1.Deployment, exists bool, err error) {
	deployment = new(v1.Deployment)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, deployment)
	return
}

// GetDeployment returns deployment with given id.
func (b *BoltDB) GetDeployment(id string) (deployment *v1.Deployment, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "Deployment")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		deployment, exists, err = b.getDeployment(id, bucket)
		return err
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *BoltDB) GetDeployments() ([]*v1.Deployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "Deployment")
	var deployments []*v1.Deployment
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deploymentBucket))
		return b.ForEach(func(k, v []byte) error {
			var deployment v1.Deployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployments = append(deployments, &deployment)
			return nil
		})
	})
	return deployments, err
}

// CountDeployments returns the number of deployments.
func (b *BoltDB) CountDeployments() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Count", "Deployment")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deploymentBucket))
		return b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})

	return
}

// AddDeployment adds a deployment to bolt
func (b *BoltDB) AddDeployment(deployment *v1.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "Deployment")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		_, exists, err := b.getDeployment(deployment.Id, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Deployment %v cannot be added because it already exists", deployment.GetId())
		}
		bytes, err := proto.Marshal(deployment)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(deployment.Id), bytes)
	})
}

func (b *BoltDB) updateDeployment(deployment *v1.Deployment, bucket *bolt.Bucket) error {
	bytes, err := proto.Marshal(deployment)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(deployment.Id), bytes)
}

// UpdateDeployment updates a deployment to bolt
func (b *BoltDB) UpdateDeployment(deployment *v1.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Update", "Deployment")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		existingDeployment, exists, err := b.getDeployment(deployment.GetId(), bucket)
		if err != nil {
			return err
		}
		// Apply the tombstone to the update. This update should have more up to date info so worth saving
		if exists && existingDeployment.GetTombstone() != nil {
			deployment.Tombstone = existingDeployment.GetTombstone()
		}
		return b.updateDeployment(deployment, tx.Bucket([]byte(deploymentBucket)))
	})
}

// RemoveDeployment updates a deployment with a tombstone
func (b *BoltDB) RemoveDeployment(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "Deployment")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		deployment, exists, err := b.getDeployment(id, bucket)
		if err != nil {
			return err
		}
		if !exists {
			return db.ErrNotFound{Type: "Deployment", ID: id}
		}
		deployment.Tombstone = ptypes.TimestampNow()
		return b.updateDeployment(deployment, bucket)
	})
}
