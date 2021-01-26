package m53tom54

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/features"
	bolt "go.etcd.io/bbolt"
)

var (
	clustersBucket = []byte("clusters")
)

var (
	migration = types.Migration{
		StartingSeqNum: 53,
		VersionAfter:   storage.Version{SeqNum: 54},
		Run: func(databases *types.Databases) error {
			return migrateExecWebhook(databases.BoltDB)
		},
	}
)

func migrateExecWebhook(db *bolt.DB) error {
	if !features.K8sEventDetection.Enabled() {
		return nil
	}

	var clustersToMigrate []*storage.Cluster // Should be able to hold all policies in memory easily
	err := db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			cluster := &storage.Cluster{}
			if err := proto.Unmarshal(v, cluster); err != nil {
				// If anything fails to unmarshal roll back the transaction and abort
				return errors.Wrapf(err, "Failed to unmarshal cluster data for key %s", k)
			}
			if cluster.GetType() == storage.ClusterType_OPENSHIFT_CLUSTER {
				return nil
			}
			cluster.AdmissionControllerEvents = true
			clustersToMigrate = append(clustersToMigrate, cluster)
			return nil
		})
	})

	if err != nil {
		return errors.Wrap(err, "reading cluster data")
	}

	if len(clustersToMigrate) == 0 {
		return nil // nothing to do
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		if bucket == nil {
			return nil
		}

		// Store successfully migrated policies.  We don't need to change the name/ID cross index.
		for _, cluster := range clustersToMigrate {
			if err := storeCluster(cluster, bucket); err != nil {
				return err
			}
		}
		return nil
	})
}

func storeCluster(cluster *storage.Cluster, bucket *bolt.Bucket) error {
	bytes, err := proto.Marshal(cluster)
	if err != nil {
		// If anything fails to marshal roll back the transaction and abort
		return errors.Wrapf(err, "failed to marshal migrated cluster %s:%s", cluster.GetName(), cluster.GetId())
	}
	// No need to update secondary mappings, we haven't changed the name and the name mapping just references the ID.
	if err := bucket.Put([]byte(cluster.GetId()), bytes); err != nil {
		return err
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
