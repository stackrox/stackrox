package store

import (
	"errors"
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/secondarykey"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	log = logging.LoggerForModule()
)

type storeImpl struct {
	*bolt.DB
}

// GetCluster returns cluster with given id.
func (b *storeImpl) GetCluster(id string) (cluster *storage.Cluster, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Get, "Cluster")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		cluster, exists, err = b.getCluster(tx, id, bucket)
		return err
	})
	return
}

// GetClusters retrieves clusters matching the request from bolt
func (b *storeImpl) GetClusters() ([]*storage.Cluster, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Cluster")
	var clusters []*storage.Cluster
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		return bucket.ForEach(func(k, v []byte) error {
			var cluster storage.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return err
			}
			b.populateClusterStatus(tx, &cluster)
			clusters = append(clusters, &cluster)
			return nil
		})
	})
	return clusters, err
}

// CountClusters returns the number of clusters.
func (b *storeImpl) CountClusters() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Count, "Cluster")
	err = b.View(func(tx *bolt.Tx) error {
		count = tx.Bucket(clusterBucket).Stats().KeyN
		return nil
	})

	return
}

// AddCluster adds a cluster to bolt
func (b *storeImpl) AddCluster(cluster *storage.Cluster) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Add, "Cluster")
	if cluster.GetId() != "" {
		return "", fmt.Errorf("cannot add a cluster that has already been assigned an id: %q", cluster.GetId())
	}
	if cluster == nil {
		return "", errors.New("cannot add a nil cluster")
	}
	cluster.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		currCluster, exists, err := b.getCluster(tx, cluster.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return errorhelpers.Newf(errorhelpers.ErrAlreadyExists,
				"Cluster %v (%v) cannot be added because it already exists", currCluster.GetId(), currCluster.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, clusterBucket, cluster.GetId(), cluster.GetName()); err != nil {
			return errorhelpers.Newf(errorhelpers.ErrAlreadyExists, "Could not add cluster due to name validation: %s", err)
		}
		bytes, err := proto.Marshal(cluster)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(cluster.GetId()), bytes)
	})

	return cluster.Id, err
}

func (b *storeImpl) updateCluster(tx *bolt.Tx, cluster *storage.Cluster) error {
	bucket := tx.Bucket(clusterBucket)
	// If the update is changing the name, check if the name has already been taken
	if val, _ := secondarykey.GetCurrentUniqueKey(tx, clusterBucket, cluster.GetId()); val != cluster.GetName() {
		if err := secondarykey.UpdateUniqueKey(tx, clusterBucket, cluster.GetId(), cluster.GetName()); err != nil {
			return fmt.Errorf("Could not update cluster due to name validation: %s", err)
		}
	}
	bytes, err := proto.Marshal(cluster)
	if err != nil {
		return err
	}
	return bucket.Put([]byte(cluster.GetId()), bytes)
}

// UpdateCluster updates a cluster to bolt
func (b *storeImpl) UpdateCluster(cluster *storage.Cluster) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Cluster")
	return b.Update(func(tx *bolt.Tx) error {
		return b.updateCluster(tx, cluster)
	})
}

// RemoveCluster removes a cluster.
// TODO(viswa): Remove from all buckets.
func (b *storeImpl) RemoveCluster(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Remove, "Cluster")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(clusterBucket)
		key := []byte(id)
		if err := secondarykey.RemoveUniqueKey(tx, clusterBucket, id); err != nil {
			return err
		}
		if exists := b.Get(key) != nil; !exists {
			return dberrors.ErrNotFound{Type: "Cluster", ID: string(key)}
		}
		if err := b.Delete(key); err != nil {
			return err
		}
		clusterStatusB := tx.Bucket(clusterStatusBucket)
		if clusterStatusBucket != nil {
			if err := clusterStatusB.Delete(key); err != nil {
				return err
			}
		}
		clusterLastContactTimeB := tx.Bucket(clusterLastContactTimeBucket)
		if clusterLastContactTimeB != nil {
			if err := clusterLastContactTimeB.Delete(key); err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateClusterContactTime stores the time at which the cluster has last
// talked with Central. This is maintained separately to avoid being clobbered
// by updates to the main Cluster config object.
func (b *storeImpl) UpdateClusterContactTime(id string, t time.Time) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "ClusterContactTime")

	tsProto, err := ptypes.TimestampProto(t)
	if err != nil {
		return err
	}
	bytes, err := proto.Marshal(tsProto)
	if err != nil {
		return err
	}

	key := []byte(id)
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterLastContactTimeBucket)
		return bucket.Put(key, bytes)
	})
}

func (b *storeImpl) getCluster(tx *bolt.Tx, id string, bucket *bolt.Bucket) (cluster *storage.Cluster, exists bool, err error) {
	cluster = new(storage.Cluster)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, cluster)
	if err != nil {
		return
	}
	b.populateClusterStatus(tx, cluster)
	return
}

func (b *storeImpl) populateClusterStatus(tx *bolt.Tx, cluster *storage.Cluster) {
	status, err := b.getClusterStatus(tx, cluster.GetId())
	if err != nil {
		log.Warnf("Could not get cluster status for %q: %v", cluster.GetId(), err)
		return
	}
	t, err := b.getClusterContactTime(tx, cluster.GetId())
	if err != nil {
		log.Warnf("Could not get cluster last-contact time for '%s': %s", cluster.GetId(), err)
		return
	}
	if t != nil {
		if status == nil {
			status = &storage.ClusterStatus{}
		}
		status.LastContact = t
	}
	cluster.Status = status
}

func (b *storeImpl) getClusterContactTime(tx *bolt.Tx, id string) (*timestamp.Timestamp, error) {
	bucket := tx.Bucket(clusterLastContactTimeBucket)
	val := bucket.Get([]byte(id))
	if val == nil {
		return nil, nil
	}
	t := new(timestamp.Timestamp)
	err := proto.Unmarshal(val, t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (b *storeImpl) getClusterStatus(tx *bolt.Tx, id string) (*storage.ClusterStatus, error) {
	bucket := tx.Bucket(clusterStatusBucket)
	val := bucket.Get([]byte(id))
	if val == nil {
		return nil, nil
	}
	status := new(storage.ClusterStatus)
	err := proto.Unmarshal(val, status)
	if err != nil {
		return nil, err
	}
	return status, nil
}

func (b *storeImpl) UpdateClusterStatus(id string, status *storage.ClusterStatus) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "ClusterStatus")
	bytes, err := proto.Marshal(status)
	if err != nil {
		return fmt.Errorf("marshaling cluster status: %v", err)
	}
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterStatusBucket)
		return bucket.Put([]byte(id), bytes)
	})
}
