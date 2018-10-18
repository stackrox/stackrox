package store

import (
	"time"

	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type storeImpl struct {
	*bolt.DB
	ranker *ranking.Ranker
}

// GetListDeployment returns a list deployment with given id.
func (b *storeImpl) ListDeployment(id string) (deployment *v1.ListDeployment, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ListDeployment")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentListBucket))
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		deployment = new(v1.ListDeployment)
		err := proto.Unmarshal(val, deployment)
		if err != nil {
			return err
		}
		deployment.Priority = b.ranker.Get(deployment.GetId())
		return nil
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) ListDeployments() ([]*v1.ListDeployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListDeployment")
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Deployment")
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Deployment")
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
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "Deployment")
	err = b.View(func(tx *bolt.Tx) error {
		count = tx.Bucket([]byte(deploymentBucket)).Stats().KeyN
		return nil
	})

	return
}

func (b *storeImpl) upsertDeployment(deployment *v1.Deployment, bucket *bolt.Bucket) error {
	bytes, err := proto.Marshal(deployment)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(deployment.Id), bytes)
}

// This needs to be called within an Update transaction, and is the common code between
// upsert and update.
func (b *storeImpl) putDeployment(deployment *v1.Deployment, tx *bolt.Tx, errorIfNotExists bool) error {
	bucket := tx.Bucket([]byte(deploymentBucket))
	_, exists, err := b.getDeployment(deployment.GetId(), bucket)
	if err != nil {
		return err
	}
	if errorIfNotExists && !exists {
		return dberrors.ErrNotFound{Type: "Deployment", ID: deployment.GetId()}
	}

	b.ranker.Add(deployment.GetId(), deployment.GetRisk().GetScore())
	if err := b.upsertDeployment(deployment, bucket); err != nil {
		return err
	}
	return b.upsertListDeployment(tx.Bucket([]byte(deploymentListBucket)), deployment)
}

// UpsertDeployment adds a deployment to bolt, or updates it if it exists already.
func (b *storeImpl) UpsertDeployment(deployment *v1.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Deployment")
	return b.Update(func(tx *bolt.Tx) error {
		return b.putDeployment(deployment, tx, false)
	})
}

// UpdateDeployment updates a deployment to bolt
func (b *storeImpl) UpdateDeployment(deployment *v1.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Deployment")
	return b.Update(func(tx *bolt.Tx) error {
		return b.putDeployment(deployment, tx, true)
	})
}

// RemoveDeployment removes a deployment
func (b *storeImpl) RemoveDeployment(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Deployment")

	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(deploymentBucket))

		b.ranker.Remove(id)
		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}
		return b.removeListDeployment(tx, id)
	})

	if err != nil {
		return err
	}
	return nil
}
