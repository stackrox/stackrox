package m53tom54

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
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
	return db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(clustersBucket)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			var cluster storage.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return err
			}
			if cluster.GetType() == storage.ClusterType_OPENSHIFT_CLUSTER {
				return nil
			}

			cluster.AdmissionControllerEvents = true
			newValue, err := proto.Marshal(&cluster)
			if err != nil {
				return errors.Wrapf(err, "error marshalling cluster %s", k)
			}
			return bucket.Put(k, newValue)
		})
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
