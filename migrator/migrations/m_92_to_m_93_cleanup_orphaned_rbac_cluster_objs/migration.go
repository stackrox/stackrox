package m92tom93

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	serviceAccountBucket = []byte("service_accounts")
	clusterBucket        = []byte("clusters")
	k8sRoleBucket        = []byte("k8sroles")
	roleBindingsBucket   = []byte("rolebindings")

	migration = types.Migration{
		StartingSeqNum: 92,
		VersionAfter:   &storage.Version{SeqNum: 93},
		Run:            cleanupOrphanedRBACObjectsFromDeletedClusters,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func cleanupOrphanedRBACObjectsFromDeletedClusters(db *types.Databases) error {
	clusters, err := getActiveClusters(db)
	if err != nil {
		return errors.Wrap(err, "listing active clusters")
	}

	deletedSACount, err := removedOrphanedObjects(db, serviceAccountBucket, func(key, data []byte) (bool, error) {
		var sa storage.ServiceAccount
		if err := proto.Unmarshal(data, &sa); err != nil {
			return false, errors.Wrapf(err, "unmarshaling service account %s", key)
		}
		return !clusters.Contains(sa.GetClusterId()), nil
	})
	if err != nil {
		return errors.Wrap(err, "deleting service accounts that don't belong to a valid cluster")
	}
	log.WriteToStderrf("Removed %d service accounts that don't belong to a valid cluster", deletedSACount)

	deletedK8SRoleCount, err := removedOrphanedObjects(db, k8sRoleBucket, func(key, data []byte) (bool, error) {
		var role storage.K8SRole
		if err := proto.Unmarshal(data, &role); err != nil {
			return false, errors.Wrapf(err, "unmarshaling K8S roles %s", key)
		}
		return !clusters.Contains(role.GetClusterId()), nil
	})
	if err != nil {
		return errors.Wrap(err, "deleting K8S roles that don't belong to a valid cluster")
	}
	log.WriteToStderrf("Removed %d K8S roles that don't belong to a valid cluster", deletedK8SRoleCount)

	deletedRoleBindingsCount, err := removedOrphanedObjects(db, roleBindingsBucket, func(key, data []byte) (bool, error) {
		var role storage.K8SRoleBinding
		if err := proto.Unmarshal(data, &role); err != nil {
			return false, errors.Wrapf(err, "unmarshaling K8S role bindings %s", key)
		}
		return !clusters.Contains(role.GetClusterId()), nil
	})
	if err != nil {
		return errors.Wrap(err, "deleting K8S role bindings that don't belong to a valid cluster")
	}
	log.WriteToStderrf("Removed %d K8S role bindings that don't belong to a valid cluster", deletedRoleBindingsCount)

	return nil
}

func removedOrphanedObjects(db *types.Databases, bucket []byte, checkIfShouldDelete func(key []byte, data []byte) (bool, error)) (int, error) {
	it := db.RocksDB.NewIterator(readOpts)
	defer it.Close()

	prefix := rocksdbmigration.GetBucketPrefix(bucket)

	var deletedCount int
	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key().Copy()

		shouldDelete, err := checkIfShouldDelete(key, it.Value().Data())
		if err != nil {
			return 0, err
		}

		if !shouldDelete {
			continue
		}

		wb.Delete(key)
		deletedCount++

		if wb.Count() == batchSize {
			if err := db.RocksDB.Write(writeOpts, wb); err != nil {
				return deletedCount, errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}

	if wb.Count() != 0 {
		if err := db.RocksDB.Write(writeOpts, wb); err != nil {
			return deletedCount, errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return deletedCount, nil
}

func getActiveClusters(db *types.Databases) (set.FrozenStringSet, error) {
	var clusters []string

	// Adding in the separator to avoid this picking up anything in clusters_health_bucket
	prefix := rocksdbmigration.GetBucketPrefix(clusterBucket)

	it := db.RocksDB.NewIterator(readOpts)
	defer it.Close()

	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key().Copy()
		clusterID := rocksdbmigration.GetIDFromPrefixedKey(clusterBucket, key)
		clusters = append(clusters, string(clusterID))
	}

	return set.NewFrozenStringSet(clusters...), nil
}
