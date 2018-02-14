package boltdb

import (
	"fmt"
	"time"

	"bitbucket.org/stack-rox/apollo/central/db"
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/uuid"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	clusterBucket       = "clusters"
	clusterStatusBucket = "clusters_status"
)

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

// GetCluster returns cluster with given id.
func (b *BoltDB) GetCluster(id string) (cluster *v1.Cluster, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterBucket))
		cluster, exists, err = b.getCluster(id, bucket)
		return err
	})
	return
}

// GetClusters retrieves clusters matching the request from bolt
func (b *BoltDB) GetClusters() ([]*v1.Cluster, error) {
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

// AddCluster adds a cluster to bolt
func (b *BoltDB) AddCluster(cluster *v1.Cluster) (string, error) {
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
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		bytes, err := proto.Marshal(cluster)
		if err != nil {
			return err
		}
		return b.Put([]byte(cluster.GetId()), bytes)
	})
}

// RemoveCluster removes a cluster.
func (b *BoltDB) RemoveCluster(id string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		key := []byte(id)
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
