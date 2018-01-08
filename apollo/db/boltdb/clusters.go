package boltdb

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const clusterBucket = "clusters"

func (b *BoltDB) getCluster(name string, bucket *bolt.Bucket) (cluster *v1.Cluster, exists bool, err error) {
	cluster = new(v1.Cluster)
	val := bucket.Get([]byte(name))
	if val == nil {
		return
	}
	exists = true
	err = proto.Unmarshal(val, cluster)
	return
}

// GetCluster returns cluster with given name.
func (b *BoltDB) GetCluster(name string) (cluster *v1.Cluster, exists bool, err error) {
	err = b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterBucket))
		cluster, exists, err = b.getCluster(name, bucket)
		return err
	})
	return
}

// GetClusters retrieves clusters matching the request from bolt
func (b *BoltDB) GetClusters() ([]*v1.Cluster, error) {
	var clusters []*v1.Cluster
	err := b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		b.ForEach(func(k, v []byte) error {
			var cluster v1.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return err
			}
			clusters = append(clusters, &cluster)
			return nil
		})
		return nil
	})
	return clusters, err
}

// AddCluster adds a cluster to bolt
func (b *BoltDB) AddCluster(cluster *v1.Cluster) error {
	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(clusterBucket))
		currCluster, exists, err := b.getCluster(cluster.Name, bucket)
		if err != nil {
			return err
		}
		if exists {
			return fmt.Errorf("Cluster %v cannot be added because it already exists", currCluster.GetName())
		}
		bytes, err := proto.Marshal(cluster)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(cluster.Name), bytes)
	})
}

// UpdateCluster updates a cluster to bolt
func (b *BoltDB) UpdateCluster(cluster *v1.Cluster) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		bytes, err := proto.Marshal(cluster)
		if err != nil {
			return err
		}
		return b.Put([]byte(cluster.Name), bytes)
	})
}

// RemoveCluster removes a cluster.
func (b *BoltDB) RemoveCluster(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		return b.Delete([]byte(name))
	})
}
