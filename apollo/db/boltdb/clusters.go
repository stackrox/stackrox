package boltdb

import (
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

const clusterBucket = "clusters"

// GetCluster returns cluster with given name.
func (b *BoltDB) GetCluster(name string) (cluster *v1.Cluster, exists bool, err error) {
	cluster = new(v1.Cluster)
	err = b.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		val := b.Get([]byte(name))
		if val == nil {
			return nil
		}
		exists = true
		return proto.Unmarshal(val, cluster)
	})

	return
}

// GetClusters retrieves clusters from Bolt.
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

func (b *BoltDB) upsertCluster(cluster *v1.Cluster) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		bytes, err := proto.Marshal(cluster)
		if err != nil {
			return err
		}
		err = b.Put([]byte(cluster.Name), bytes)
		return err
	})
}

// AddCluster adds a cluster to bolt
func (b *BoltDB) AddCluster(cluster *v1.Cluster) error {
	return b.upsertCluster(cluster)
}

// UpdateCluster updates a cluster to bolt
func (b *BoltDB) UpdateCluster(cluster *v1.Cluster) error {
	return b.upsertCluster(cluster)
}

// RemoveCluster removes a cluster.
func (b *BoltDB) RemoveCluster(name string) error {
	return b.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(clusterBucket))
		return b.Delete([]byte(name))
	})
}
