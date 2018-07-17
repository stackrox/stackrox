package store

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/central/ranking"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/dberrors"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
)

type storeImpl struct {
	*bolt.DB
	ranker *ranking.Ranker
}

// GetListDeployment returns a list deployment with given id.
func (b *storeImpl) ListDeployment(id string) (deployment *v1.ListDeployment, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "ListDeployment")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentListBucket))
		deployment = new(v1.ListDeployment)
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		deployment.Priority = b.ranker.Get(deployment.GetId())
		err = proto.Unmarshal(val, deployment)
		return nil
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) ListDeployments() ([]*v1.ListDeployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "ListDeployment")
	var deployments []*v1.ListDeployment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentListBucket))
		return bucket.ForEach(func(k, v []byte) error {
			var deployment v1.ListDeployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployment.Priority = b.ranker.Get(deployment.GetId())
			deployments = append(deployments, &deployment)
			return nil
		})
	})
	return deployments, err
}

// Note: This is called within a txn and do not require an Update or View
func (b *storeImpl) upsertListDeployment(bucket *bolt.Bucket, deployment *v1.Deployment) error {
	listDeployment := convertDeploymentToDeploymentList(deployment)
	bytes, err := proto.Marshal(listDeployment)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(deployment.Id), bytes)
}

// Note: This is called within a txn and do not require an Update or View
func (b *storeImpl) removeListDeployment(tx *bolt.Tx, id string) error {
	bucket := tx.Bucket([]byte(deploymentListBucket))
	return bucket.Delete([]byte(id))
}

func convertDeploymentToDeploymentList(d *v1.Deployment) *v1.ListDeployment {
	return &v1.ListDeployment{
		Id:        d.GetId(),
		Name:      d.GetName(),
		Cluster:   d.GetClusterName(),
		ClusterId: d.GetClusterId(),
		Namespace: d.GetNamespace(),
		UpdatedAt: d.GetUpdatedAt(),
		Priority:  d.GetPriority(),
	}
}

func (b *storeImpl) getDeployment(id string, bucket *bolt.Bucket) (deployment *v1.Deployment, exists bool, err error) {
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
func (b *storeImpl) GetDeployment(id string) (deployment *v1.Deployment, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "Deployment")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		deployment, exists, err = b.getDeployment(id, bucket)
		if err != nil {
			return err
		}
		if exists {
			deployment.Priority = b.ranker.Get(id)
		}

		return nil
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) GetDeployments() ([]*v1.Deployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "Deployment")
	var deployments []*v1.Deployment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		return bucket.ForEach(func(k, v []byte) error {
			var deployment v1.Deployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployment.Priority = b.ranker.Get(deployment.GetId())

			deployments = append(deployments, &deployment)
			return nil
		})
	})
	return deployments, err
}

// CountDeployments returns the number of deployments.
func (b *storeImpl) CountDeployments() (count int, err error) {
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
func (b *storeImpl) AddDeployment(deployment *v1.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "Deployment")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))
		_, exists, err := b.getDeployment(deployment.Id, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("deployment %s cannot be added because it already exists", deployment.GetId())
		}
		b.ranker.Add(deployment.GetId(), deployment.GetRisk().GetScore())
		bytes, err := proto.Marshal(deployment)
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte(deployment.Id), bytes); err != nil {
			return err
		}
		return b.upsertListDeployment(tx.Bucket([]byte(deploymentListBucket)), deployment)
	})
}

func (b *storeImpl) updateDeployment(deployment *v1.Deployment, bucket *bolt.Bucket) error {
	bytes, err := proto.Marshal(deployment)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(deployment.Id), bytes)
}

// UpdateDeployment updates a deployment to bolt
func (b *storeImpl) UpdateDeployment(deployment *v1.Deployment) error {
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
		b.ranker.Add(deployment.GetId(), deployment.GetRisk().GetScore())
		if err := b.updateDeployment(deployment, tx.Bucket([]byte(deploymentBucket))); err != nil {
			return err
		}
		return b.upsertListDeployment(tx.Bucket([]byte(deploymentListBucket)), deployment)
	})
}

// RemoveDeployment updates a deployment with a tombstone
func (b *storeImpl) RemoveDeployment(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "Deployment")

	var deployment *v1.Deployment
	var exists bool
	var err error
	b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))

		deployment, exists, err = b.getDeployment(id, bucket)
		if err != nil {
			return err
		}
		if !exists {
			return dberrors.ErrNotFound{Type: "Deployment", ID: id}
		}
		b.ranker.Remove(id)
		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}
		return b.removeListDeployment(tx, id)
	})

	if deployment != nil {
		b.addDeploymentToGraveyard(deployment)
	}
	return err
}

// GetTombstonedDeployments returns all of the deployments that have been tombstoned.
func (b *storeImpl) GetTombstonedDeployments() ([]*v1.Deployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "TombstonedDeployment")
	var deployments []*v1.Deployment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentGraveyard))
		return bucket.ForEach(func(k, v []byte) error {
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

func (b *storeImpl) addDeploymentToGraveyard(deployment *v1.Deployment) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "TombstonedDeployment")
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentGraveyard))

		if val := bucket.Get([]byte(deployment.GetId())); val != nil {
			return fmt.Errorf("deployment %s cannot be tombstoned because it already has been", deployment.GetId())
		}

		deployment.Tombstone = ptypes.TimestampNow()
		bytes, err := proto.Marshal(deployment)
		if err != nil {
			return err
		}

		return bucket.Put([]byte(deployment.GetId()), bytes)
	})

	// If there is an error stuffing a deployment in the graveyard, then just abandon it in the street.
	if err != nil {
		log.Errorf("unable to tombstone deployment", err)
	}
}
