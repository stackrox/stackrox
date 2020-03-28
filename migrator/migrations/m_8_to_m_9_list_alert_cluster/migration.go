package m8to9

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/bolthelpers"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
)

var migration = types.Migration{
	StartingSeqNum: 8,
	VersionAfter:   storage.Version{SeqNum: 9},
	Run:            upgradeAlertsForSAC,
}

func init() {
	migrations.MustRegisterMigration(migration)
}

func upgradeAlertsForSAC(db *bolt.DB, _ *badger.DB) error {
	if err := upgradeAllListAlerts(db); err != nil {
		return err
	}
	return nil
}

var (
	clusterBucketName   = []byte("clusters")
	listAlertBucketName = []byte("alerts_list")
)

func upgradeAllListAlerts(db *bolt.DB) error {
	// Create a map of alert ID to cluster ID.
	clusterNameToClusterID := make(map[string]string)
	clusterBucket := bolthelpers.TopLevelRef(db, clusterBucketName)
	err := clusterBucket.View(func(b *bolt.Bucket) error {
		return b.ForEach(func(_, v []byte) error {
			cluster := &storage.Cluster{}
			if err := proto.Unmarshal(v, cluster); err != nil {
				return err
			}
			clusterNameToClusterID[cluster.GetName()] = cluster.GetId()
			return nil
		})
	})
	if err != nil {
		return err
	}

	// Insert those values into the list alerts.
	listAlertBucket := bolthelpers.TopLevelRef(db, listAlertBucketName)
	return listAlertBucket.Update(func(b *bolt.Bucket) error {
		return b.ForEach(func(k, v []byte) error {
			// Get current ListAlert value.
			listAlert := &storage.ListAlert{}
			if err := proto.Unmarshal(v, listAlert); err != nil {
				return errors.Wrap(err, "unmarshaling ListAlert")
			}

			// Add the cluster id to it.
			listAlert.Deployment.ClusterId = clusterNameToClusterID[listAlert.GetDeployment().GetClusterName()]

			// Remarshal and save the new value.
			bytes, err := proto.Marshal(listAlert)
			if err != nil {
				return err
			}
			return b.Put(k, bytes)
		})
	})
}
