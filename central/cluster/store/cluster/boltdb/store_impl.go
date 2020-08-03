package boltdb

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/cluster/store"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/bolthelper"
	ops "github.com/stackrox/rox/pkg/metrics"
)

var (
	clusterBucket = []byte("clusters")
)

type storeImpl struct {
	*bolt.DB
}

// New returns a new ClusterStore instance using the provided bolt DB instance.
func New(db *bolt.DB) store.ClusterStore {
	bolthelper.RegisterBucketOrPanic(db, clusterBucket)
	return &storeImpl{
		DB: db,
	}
}

// Get returns cluster with given id.
func (b *storeImpl) Get(id string) (cluster *storage.Cluster, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Cluster")
	return nil, false, errors.New("no longer implemented")
}

// Walk retrieves clusters matching the request from bolt
func (b *storeImpl) Walk(fn func(cluster *storage.Cluster) error) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Cluster")
	return errors.New("no longer implemented")
}

// GetMany retrieves clusters with the given IDs from bolt.
func (b *storeImpl) GetMany(ids []string) ([]*storage.Cluster, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Cluster")
	return nil, nil, errors.New("no longer implemented")
}

// Count returns the number of clusters.
func (b *storeImpl) Count() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "Cluster")
	return 0, errors.New("no longer implemented")
}

// UpdateCluster updates a cluster to bolt
func (b *storeImpl) Upsert(cluster *storage.Cluster) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Cluster")
	return errors.New("no longer implemented")
}

// Delete removes a cluster.
func (b *storeImpl) Delete(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Cluster")
	return errors.New("no longer implemented")
}
