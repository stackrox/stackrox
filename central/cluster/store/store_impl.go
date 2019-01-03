package store

import (
	"fmt"
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/dberrors"
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
			b.populateProtoContactTime(tx, &cluster)
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
	cluster.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		currCluster, exists, err := b.getCluster(tx, cluster.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Cluster %v (%v) cannot be added because it already exists", currCluster.GetId(), currCluster.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, clusterBucket, cluster.GetId(), cluster.GetName()); err != nil {
			return fmt.Errorf("Could not add cluster due to name validation: %s", err)
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
		return b.Delete(key)
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
	b.populateProtoContactTime(tx, cluster)
	return
}

func (b *storeImpl) populateProtoContactTime(tx *bolt.Tx, cluster *storage.Cluster) {
	t, err := b.getClusterContactTime(tx, cluster.GetId())
	if err != nil {
		log.Warnf("Could not get cluster last-contact time for '%s': %s", cluster.GetId(), err)
		return
	}
	cluster.LastContact = t
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

// UpdateMetadata updates the cluster with cloud provider metadata
func (b *storeImpl) UpdateMetadata(id string, metadata *storage.ProviderMetadata) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, "Cluster")

	return b.Update(func(tx *bolt.Tx) error {
		cluster, exists, err := b.getCluster(tx, id, tx.Bucket(clusterBucket))
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("could not enrich cluster with metadata. Cluster %q does not exist", id)
		}
		cluster.ProviderMetadata = metadata
		return b.updateCluster(tx, cluster)
	})
}
