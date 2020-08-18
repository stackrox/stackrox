package m44tom45

import (
	"time"

	"github.com/gogo/protobuf/proto"
	pTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 44,
		VersionAfter:   storage.Version{SeqNum: 45},
		Run:            migrateClusterBuckets,
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

var (
	clusterBucketName                = []byte("clusters")
	clusterStatusBucketName          = []byte("cluster_status")
	clusterLastContactTimeBucketName = []byte("clusters_last_contact")
	// newly added on rocksDB
	clusterHealthStatusBucketName = []byte("clusters_health_status")
)

func migrateClusterBuckets(databases *types.Databases) error {
	// Merge cluster status bucket into cluster bucket and migrate.
	count, err := mergeAndMigrateClusterAndStatus(databases.BoltDB, databases.RocksDB)
	if err != nil {
		return err
	}
	log.WriteToStderrf("Rewrote %d keys from Bolt Bucket %s and %s", count, clusterBucketName, clusterStatusBucketName)

	// Migrate last contact cluster_status bucket into new cluster health bucket.
	count, err = migrateLastContact(databases.BoltDB, databases.RocksDB)
	if err != nil {
		return err
	}
	log.WriteToStderrf("Rewrote %d keys from Bolt Bucket %s", count, clusterLastContactTimeBucketName)
	return nil
}

func mergeAndMigrateClusterAndStatus(boltDB *bbolt.DB, rocksDB *gorocksdb.DB) (int, error) {
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	var count int
	err := boltDB.View(func(tx *bbolt.Tx) error {
		clusterBucket := tx.Bucket(clusterBucketName)
		if clusterBucket == nil {
			return nil
		}

		clusterStatusBucket := tx.Bucket(clusterStatusBucketName)
		if clusterStatusBucket == nil {
			return nil
		}

		return clusterBucket.ForEach(func(k, v []byte) error {
			newKey := rocksdbmigration.GetPrefixedKey(clusterBucketName, k)
			var cluster storage.Cluster
			if err := proto.Unmarshal(v, &cluster); err != nil {
				return err
			}

			// Merge status into cluster.
			statusValue := clusterStatusBucket.Get(k)
			if statusValue != nil {
				var clusterStatus storage.ClusterStatus
				if err := proto.Unmarshal(statusValue, &clusterStatus); err != nil {
					return err
				}
				cluster.Status = &clusterStatus
			}

			newValue, err := proto.Marshal(&cluster)
			if err != nil {
				return err
			}

			count++
			rocksWriteBatch.Put(newKey, newValue)
			return nil
		})
	})
	if err != nil {
		return 0, err
	}
	err = rocksDB.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func migrateLastContact(boltDB *bbolt.DB, rocksDB *gorocksdb.DB) (int, error) {
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	var count int
	err := boltDB.View(func(tx *bbolt.Tx) error {
		clusterLastContactBucket := tx.Bucket(clusterLastContactTimeBucketName)
		if clusterLastContactBucket == nil {
			return nil
		}

		return clusterLastContactBucket.ForEach(func(k, v []byte) error {
			newKey := rocksdbmigration.GetPrefixedKey(clusterHealthStatusBucketName, k)
			// Merge last contact into health status.
			var lastContact pTypes.Timestamp
			if err := proto.Unmarshal(v, &lastContact); err != nil {
				return err
			}

			prevContact, err := pTypes.TimestampFromProto(&lastContact)
			if err != nil {
				prevContact = time.Time{}
			}

			sensorStatus := populateSensorStatus(prevContact)
			healthStatus := storage.ClusterHealthStatus{
				SensorHealthStatus:    sensorStatus,
				OverallHealthStatus:   sensorStatus,
				CollectorHealthStatus: storage.ClusterHealthStatus_UNAVAILABLE,
				LastContact:           &lastContact,
			}

			newValue, err := proto.Marshal(&healthStatus)
			if err != nil {
				return err
			}

			count++
			rocksWriteBatch.Put(newKey, newValue)
			return nil
		})
	})
	if err != nil {
		return 0, err
	}
	err = rocksDB.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
	if err != nil {
		return 0, err
	}
	return count, nil
}
