package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const deploymentBucket = `deployments`

// GetDeployment returns deployment with given id.
func (b *BoltDB) GetDeployment(id string) (deployment *v1.Deployment, exists bool, err error) {
	deployment = new(v1.Deployment)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deploymentBucket))
		val := b.Get([]byte(id))
		if val == nil {
			exists = false
			return nil
		}

		exists = true
		return proto.Unmarshal(val, deployment)
	})

	return
}

// GetDeployments returns all deployments.
func (b *BoltDB) GetDeployments() (deployments []*v1.Deployment, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deploymentBucket))
		b.ForEach(func(k, v []byte) error {
			var deployment v1.Deployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployments = append(deployments, &deployment)
			return nil
		})
		return nil
	})

	return
}

// AddDeployment inserts a deployment.
func (b *BoltDB) AddDeployment(deployment *v1.Deployment) error {
	return b.upsertDeployment(deployment)
}

// UpdateDeployment updates a deployment.
func (b *BoltDB) UpdateDeployment(deployment *v1.Deployment) error {
	return b.upsertDeployment(deployment)
}

// RemoveDeployment removes a deployment.
func (b *BoltDB) RemoveDeployment(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deploymentBucket))
		return b.Delete([]byte(id))
	})
}

func (b *BoltDB) upsertDeployment(deployment *v1.Deployment) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(deploymentBucket))
		bytes, err := proto.Marshal(deployment)
		if err != nil {
			return err
		}
		return b.Put([]byte(deployment.Id), bytes)
	})
}
