package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/ranking"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	"github.com/stackrox/rox/pkg/dberrors"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/objects"
)

type storeImpl struct {
	*bolthelper.BoltWrapper
	ranker *ranking.Ranker
}

func (b *storeImpl) initializeRanker() error {
	b.ranker = ranking.DeploymentRanker()
	deployments, err := b.GetDeployments()
	if err != nil {
		return errors.Wrap(err, "retrieving deployments")
	}
	for _, deployment := range deployments {
		b.ranker.Add(deployment.GetId(), deployment.GetRisk().GetScore())
	}
	return nil
}

// GetListDeployment returns a list deployment with given id.
func (b *storeImpl) ListDeployment(id string) (deployment *storage.ListDeployment, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "ListDeployment")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentListBucket)
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		exists = true
		deployment = new(storage.ListDeployment)
		err := proto.Unmarshal(val, deployment)
		if err != nil {
			return err
		}
		deployment.Priority = b.ranker.GetRankForID(deployment.GetId())
		return nil
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) ListDeployments() ([]*storage.ListDeployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListDeployment")
	var deployments []*storage.ListDeployment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentListBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var deployment storage.ListDeployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployment.Priority = b.ranker.GetRankForID(deployment.GetId())
			deployments = append(deployments, &deployment)
			return nil
		})
	})
	return deployments, err
}

func (b *storeImpl) ListDeploymentsWithIDs(ids ...string) ([]*storage.ListDeployment, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "ListDeployment")
	deployments := make([]*storage.ListDeployment, 0, len(ids))
	var missingIndices []int
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentListBucket)
		for i, id := range ids {
			v := bucket.Get([]byte(id))
			if v == nil {
				missingIndices = append(missingIndices, i)
				continue
			}
			var deployment storage.ListDeployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployment.Priority = b.ranker.GetRankForID(deployment.GetId())
			deployments = append(deployments, &deployment)
		}
		return nil
	})
	return deployments, missingIndices, err
}

// Note: This is called within a txn and do not require an Update or View
func (b *storeImpl) removeListDeployment(tx *bolt.Tx, id string) error {
	bucket := tx.Bucket(deploymentListBucket)
	return bucket.Delete([]byte(id))
}

func (b *storeImpl) getDeployment(id string, bucket *bolt.Bucket) (deployment *storage.Deployment, exists bool, err error) {
	deployment = new(storage.Deployment)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, deployment)
	return
}

// GetDeployment returns deployment with given id.
func (b *storeImpl) GetDeployment(id string) (deployment *storage.Deployment, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Deployment")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentBucket)
		deployment, exists, err = b.getDeployment(id, bucket)
		if err != nil {
			return err
		}
		if exists {
			deployment.Priority = b.ranker.GetRankForID(id)
		}

		return nil
	})
	return
}

// GetDeployments retrieves deployments matching the request from bolt
func (b *storeImpl) GetDeployments() ([]*storage.Deployment, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Deployment")
	var deployments []*storage.Deployment
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var deployment storage.Deployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployment.Priority = b.ranker.GetRankForID(deployment.GetId())

			deployments = append(deployments, &deployment)
			return nil
		})
	})
	return deployments, err
}

func (b *storeImpl) GetDeploymentsWithIDs(ids ...string) ([]*storage.Deployment, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Deployment")
	deployments := make([]*storage.Deployment, 0, len(ids))
	var missingIndices []int
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentBucket)
		for i, id := range ids {
			v := bucket.Get([]byte(id))
			if v == nil {
				missingIndices = append(missingIndices, i)
				continue
			}
			var deployment storage.Deployment
			if err := proto.Unmarshal(v, &deployment); err != nil {
				return err
			}
			deployment.Priority = b.ranker.GetRankForID(deployment.GetId())

			deployments = append(deployments, &deployment)
		}
		return nil
	})
	return deployments, missingIndices, err
}

// CountDeployments returns the number of deployments.
func (b *storeImpl) CountDeployments() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "Deployment")
	err = b.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(deploymentBucket).Stats().KeyN
		return nil
	})

	return
}

func (b *storeImpl) putDeployment(deployment *storage.Deployment, errorIfNotExists bool) error {
	bytes, err := proto.Marshal(deployment)
	if err != nil {
		return err
	}

	listBytes, err := proto.Marshal(objects.ToListDeployment(deployment))
	if err != nil {
		return err
	}

	id := []byte(deployment.GetId())
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentBucket)
		if errorIfNotExists && !bolthelper.Exists(bucket, deployment.GetId()) {
			return dberrors.ErrNotFound{Type: "Deployment", ID: deployment.GetId()}
		}
		b.ranker.Add(deployment.GetId(), deployment.GetRisk().GetScore())

		if err := bucket.Put(id, bytes); err != nil {
			return err
		}

		listBucket := tx.Bucket(deploymentListBucket)
		return listBucket.Put(id, listBytes)
	})
}

// UpsertDeployment adds a deployment to bolt, or updates it if it exists already.
func (b *storeImpl) UpsertDeployment(deployment *storage.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Deployment")
	return b.putDeployment(deployment, false)
}

// UpdateDeployment updates a deployment to bolt
func (b *storeImpl) UpdateDeployment(deployment *storage.Deployment) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Deployment")
	return b.putDeployment(deployment, true)
}

// RemoveDeployment removes a deployment
func (b *storeImpl) RemoveDeployment(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Deployment")

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(deploymentBucket)

		b.ranker.Remove(id)
		if err := bucket.Delete([]byte(id)); err != nil {
			return err
		}
		return b.removeListDeployment(tx, id)
	})
}

func (b *storeImpl) GetTxnCount() (txNum uint64, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		txNum = b.BoltWrapper.GetTxnCount(tx)
		return nil
	})
	return
}

func (b *storeImpl) IncTxnCount() error {
	return b.Update(func(tx *bolt.Tx) error {
		// The b.Update increments the txn count automatically
		return nil
	})
}
