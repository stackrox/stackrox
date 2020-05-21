package m35tom36

import (
	"github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var (
	migration = types.Migration{
		StartingSeqNum: 35,
		VersionAfter:   storage.Version{SeqNum: 36},
		Run: func(databases *types.Databases) error {
			err := addDynamicClusterConfig(databases.BoltDB)
			if err != nil {
				return errors.Wrap(err, "normalizing cluster settings")
			}
			return nil
		},
	}

	clustersBucket = []byte("clusters")
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func addDynamicClusterConfig(db *bbolt.DB) error {
	clustersToMigrate := make(map[string]*storage.Cluster)

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var cluster storage.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return errors.Wrapf(err, "failed to unmarshal cluster settings for key %s", k)
			}
			if !needsNormalization(&cluster) {
				return nil // already migrated
			}
			clustersToMigrate[string(k)] = &cluster
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "reading cluster settings")
	}

	if len(clustersToMigrate) == 0 {
		return nil // nothing to do
	}

	for _, cluster := range clustersToMigrate {
		normalizeCluster(cluster)
	}

	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", clustersBucket)
		}
		for id, cluster := range clustersToMigrate {
			bytes, err := proto.Marshal(cluster)
			if err != nil {
				return errors.Errorf("failed to marshal migrated cluster settings for key %s: %v", id, err)
			}
			if err := bucket.Put([]byte(id), bytes); err != nil {
				return err
			}
		}
		return nil
	})
}
