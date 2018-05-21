package boltdb

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/central/metrics"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	timestamp "github.com/gogo/protobuf/types"
)

const (
	clusterBucket       = "clusters"
	clusterStatusBucket = "clusters_status"
)

// GetCluster returns cluster with given id.
func (b *BoltDB) GetCluster(id string) (cluster *v1.Cluster, exists bool, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Get", "Cluster")
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterBucket))
		cluster, exists, err = b.getCluster(id, bucket)
		return err
	})
	return
}

// GetClusters retrieves clusters matching the request from bolt
func (b *BoltDB) GetClusters() ([]*v1.Cluster, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "GetMany", "Cluster")
	var clusters []*v1.Cluster
	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterBucket))
		return bucket.ForEach(func(k, v []byte) error {
			var cluster v1.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return err
			}
			b.addContactTime(&cluster)
			clusters = append(clusters, &cluster)
			return nil
		})
	})
	return clusters, err
}

// CountClusters returns the number of clusters.
func (b *BoltDB) CountClusters() (count int, err error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Count", "Cluster")
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		return b.ForEach(func(k, v []byte) error {
			count++
			return nil
		})
	})

	return
}

// AddCluster adds a cluster to bolt
func (b *BoltDB) AddCluster(cluster *v1.Cluster) (string, error) {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Add", "Cluster")
	cluster.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterBucket))
		currCluster, exists, err := b.getCluster(cluster.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Cluster %v (%v) cannot be added because it already exists", currCluster.GetId(), currCluster.GetName())
		}
		if err := checkUniqueKeyExistsAndInsert(tx, clusterBucket, cluster.GetId(), cluster.GetName()); err != nil {
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

// UpdateCluster updates a cluster to bolt
func (b *BoltDB) UpdateCluster(cluster *v1.Cluster) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Update", "Cluster")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		// If the update is changing the name, check if the name has already been taken
		if getCurrentUniqueKey(tx, clusterBucket, cluster.GetId()) != cluster.GetName() {
			if err := checkUniqueKeyExistsAndInsert(tx, clusterBucket, cluster.GetId(), cluster.GetName()); err != nil {
				return fmt.Errorf("Could not update cluster due to name validation: %s", err)
			}
		}
		bytes, err := proto.Marshal(cluster)
		if err != nil {
			return err
		}
		return b.Put([]byte(cluster.GetId()), bytes)
	})
}

// RemoveCluster removes a cluster.
func (b *BoltDB) RemoveCluster(id string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "Remove", "Cluster")
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		key := []byte(id)
		if err := removeUniqueKey(tx, clusterBucket, id); err != nil {
			return err
		}
		if exists := b.Get(key) != nil; !exists {
			return db.ErrNotFound{Type: "Cluster", ID: string(key)}
		}
		return b.Delete(key)
	})
}

// UpdateClusterContactTime stores the time at which the cluster has last
// talked with Central. This is maintained separately to avoid being clobbered
// by updates to the main Cluster config object.
func (b *BoltDB) UpdateClusterContactTime(id string, t time.Time) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), "UpdateContactTime", "Cluster")
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterStatusBucket))
		tsProto, err := ptypes.TimestampProto(t)
		if err != nil {
			return err
		}
		status := &v1.ClusterStatus{
			LastContact: tsProto,
		}
		bytes, err := proto.Marshal(status)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(id), bytes)
	})
}

func (b *BoltDB) getCluster(id string, bucket *bolt.Bucket) (cluster *v1.Cluster, exists bool, err error) {
	cluster = new(v1.Cluster)
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, cluster)
	if err != nil {
		return
	}
	b.addContactTime(cluster)
	return
}

func (b *BoltDB) addContactTime(cluster *v1.Cluster) {
	t, err := b.getClusterContactTime(cluster.GetId())
	if err != nil {
		log.Warnf("Could not get cluster last-contact time for '%s': %s", cluster.GetId(), err)
		return
	}
	cluster.LastContact = t
}

func (b *BoltDB) getClusterContactTime(id string) (t *timestamp.Timestamp, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterStatusBucket))
		val := bucket.Get([]byte(id))
		if val == nil {
			return nil
		}
		status := new(v1.ClusterStatus)
		err = proto.Unmarshal(val, status)
		if err != nil {
			return err
		}
		t = status.GetLastContact()
		return nil
	})
	return
}
