package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
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
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		deployment, exists, err = b.getDeployment(id, bucket)
		return err
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *BoltDB) GetDeployments(request *v1.GetDeploymentsRequest) ([]*v1.Deployment, error) {
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

// AddDeployment adds a deployment to bolt
func (b *BoltDB) AddDeployment(deployment *v1.Deployment) error {
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
		if err := bucket.Put([]byte(deployment.Id), bytes); err != nil {
			return err
		}
		return b.indexer.AddDeployment(deployment)
	})
}

// UpdateDeployment updates a deployment to bolt
func (b *BoltDB) UpdateDeployment(deployment *v1.Deployment) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		bytes, err := proto.Marshal(deployment)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(deployment.Id), bytes); err != nil {
			return err
		}
		return b.indexer.AddDeployment(deployment)
	})
}

// RemoveDeployment removes a deployment.
func (b *BoltDB) RemoveDeployment(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		key := []byte(id)
		if exists := bucket.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Deployment", ID: string(key)}
		}
		if err := bucket.Delete(key); err != nil {
			return err
		}
		return b.indexer.DeleteDeployment(id)
	})
}
