package store

import (
	"time"

	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	ptypes "github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
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
			b.populateClusterStatus(tx, &cluster)
			clusters = append(clusters, &cluster)
			return nil
		})
	})
	return clusters, err
}

// GetSelectedClusters retrieves clusters with the given IDs from bolt.
func (b *storeImpl) GetSelectedClusters(ids []string) ([]*storage.Cluster, []int, error) {
	if len(ids) == 0 {
		return nil, nil, nil
	}

	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.GetMany, "Cluster")
	var missingIndices []int
	clusters := make([]*storage.Cluster, 0, len(ids))

	err := b.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		for i, id := range ids {
			value := bucket.Get([]byte(id))
			if value == nil {
				missingIndices = append(missingIndices, i)
				continue
			}
			var cluster storage.Cluster
			if err := proto.Unmarshal(value, &cluster); err != nil {
				return err
			}
			b.populateClusterStatus(tx, &cluster)
			clusters = append(clusters, &cluster)
		}
		return nil
	})

	if err != nil {
		return nil, nil, err
	}
	return clusters, missingIndices, nil
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
		return "", errors.Errorf("cannot add a cluster that has already been assigned an id: %q", cluster.GetId())
	}
	if cluster.GetName() == "" {
		return "", errors.New("cannot add a cluster without a name")
	}
	cluster.Id = uuid.NewV4().String()
	err := b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterBucket)
		currCluster, exists, err := b.getCluster(tx, cluster.GetId(), bucket)
		if err != nil {
			return err
		}
		if exists {
			return errors.Wrapf(ErrAlreadyExists, "could not add cluster with ID %s (%s)", currCluster.GetId(), currCluster.GetName())
		}
		if err := secondarykey.CheckUniqueKeyExistsAndInsert(tx, clusterBucket, cluster.GetId(), cluster.GetName()); err != nil {
			return errors.Wrapf(ErrAlreadyExists, "could not add cluster %s", cluster.GetName())
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
			return errors.Wrap(err, "Could not update cluster due to name validation")
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

// UpdateClusterContactTimes stores the time at which the cluster has last
// talked with Central. This is maintained separately to avoid being clobbered
// by updates to the main Cluster config object.
func (b *storeImpl) UpdateClusterContactTimes(t time.Time, ids ...string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.UpdateMany, "ClusterContactTime")

	tsProto, err := ptypes.TimestampProto(t)
	if err != nil {
		return err
	}
	bytes, err := proto.Marshal(tsProto)
	if err != nil {
		return err
	}

	return b.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clusterLastContactTimeBucket)
		for _, id := range ids {
			if err := bucket.Put([]byte(id), bytes); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *storeImpl) getCluster(tx *bolt.Tx, id string, bucket *bolt.Bucket) (cluster *storage.Cluster, exists bool, err error) {
	val := bucket.Get([]byte(id))
	if val == nil {
		return
	}
	cluster = new(storage.Cluster)
	exists = true
	err = proto.Unmarshal(val, cluster)
	if err != nil {
		return
	}
	b.populateClusterStatus(tx, cluster)
	return
}

func (b *storeImpl) populateClusterStatus(tx *bolt.Tx, cluster *storage.Cluster) {
	id := []byte(cluster.GetId())
	status, err := b.getClusterStatus(tx, id)
	if err != nil {
		log.Warnf("Could not get cluster status for %q: %v", cluster.GetId(), err)
		return
	}
	if lastContact := getClusterContactTime(tx, id); lastContact != nil {
		if status == nil {
			status = &storage.ClusterStatus{}
		}
		status.LastContact = lastContact
	}

	cluster.Status = status
}

func getClusterContactTime(tx *bolt.Tx, id []byte) *ptypes.Timestamp {
	bucket := tx.Bucket(clusterLastContactTimeBucket)
	val := bucket.Get(id)
	if val == nil {
		return nil
	}
	t := new(ptypes.Timestamp)
	err := proto.Unmarshal(val, t)
	if err != nil {
		log.Warnf("Couldn't unmarshal last contact time for cluster id %s: %v", string(id), err)
		return nil
	}
	return t
}

func (b *storeImpl) getClusterStatus(tx *bolt.Tx, id []byte) (*storage.ClusterStatus, error) {
	bucket := tx.Bucket(clusterStatusBucket)
	val := bucket.Get(id)
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
	idBytes := []byte(id)
	return b.Update(func(tx *bolt.Tx) error {
		existingStatus, err := b.getClusterStatus(tx, idBytes)
		if err != nil {
			return err
		}
		if existingStatus != nil {
			// Since all we're doing is setting status.UpgradeStatus, a shallow copy suffices.
			shallowClonedStatus := *status
			status = &shallowClonedStatus
			status.UpgradeStatus = existingStatus.UpgradeStatus
			status.CertExpiryStatus = existingStatus.CertExpiryStatus
		}

		bytes, err := proto.Marshal(status)
		if err != nil {
			return errors.Wrap(err, "marshaling cluster status")
		}
		bucket := tx.Bucket(clusterStatusBucket)
		return bucket.Put(idBytes, bytes)
	})
}

func (b *storeImpl) UpdateClusterUpgradeStatus(id string, upgradeStatus *storage.ClusterUpgradeStatus) error {
	return b.partialUpdateClusterStatus(id, func(status *storage.ClusterStatus) {
		status.UpgradeStatus = upgradeStatus
	}, "ClusterUpgradeStatus")
}

func (b *storeImpl) UpdateClusterCertExpiryStatus(id string, certExpiryStatus *storage.ClusterCertExpiryStatus) error {
	return b.partialUpdateClusterStatus(id, func(status *storage.ClusterStatus) {
		status.CertExpiryStatus = certExpiryStatus
	}, "ClusterCertExpiryStatus")
}

func (b *storeImpl) partialUpdateClusterStatus(id string, partialUpdateFunc func(*storage.ClusterStatus), opType string) error {
	defer metrics.SetBoltOperationDurationTime(time.Now(), ops.Update, opType)
	idBytes := []byte(id)
	return b.Update(func(tx *bolt.Tx) error {
		existingStatus, err := b.getClusterStatus(tx, idBytes)
		if err != nil {
			return err
		}
		if existingStatus == nil {
			existingStatus = new(storage.ClusterStatus)
		}
		partialUpdateFunc(existingStatus)
		bytes, err := proto.Marshal(existingStatus)
		if err != nil {
			return errors.Wrap(err, "marshaling cluster status")
		}
		bucket := tx.Bucket(clusterStatusBucket)
		return bucket.Put(idBytes, bytes)
	})

}
