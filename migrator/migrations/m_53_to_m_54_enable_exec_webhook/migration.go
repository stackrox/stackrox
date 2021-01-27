package m53tom54

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/features"
	"github.com/tecbot/gorocksdb"
)

var (
	clustersPrefix = []byte("clusters")
	separator      = []byte("\x00")
)

var (
	migration = types.Migration{
		StartingSeqNum: 53,
		VersionAfter:   storage.Version{SeqNum: 54},
		Run: func(databases *types.Databases) error {
			return migrateExecWebhook(databases.RocksDB)
		},
	}
)

func migrateExecWebhook(db *gorocksdb.DB) error {
	if !features.K8sEventDetection.Enabled() {
		return nil
	}

	var clustersToMigrate []*storage.Cluster // Should be able to hold all policies in memory easily
	readOpts := gorocksdb.NewDefaultReadOptions()
	it := db.NewIterator(readOpts)
	defer it.Close()

	prefix := getPrefix()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		cluster := &storage.Cluster{}
		if err := proto.Unmarshal(it.Value().Data(), cluster); err != nil {
			// If anything fails to unmarshal roll back the transaction and abort
			return errors.Wrapf(err, "Failed to unmarshal cluster data for key %v", it.Key().Data())
		}
		if cluster.GetType() == storage.ClusterType_OPENSHIFT_CLUSTER {
			continue
		}
		cluster.AdmissionControllerEvents = true
		clustersToMigrate = append(clustersToMigrate, cluster)
	}

	if len(clustersToMigrate) == 0 {
		return nil // nothing to do
	}
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	for _, c := range clustersToMigrate {
		bytes, err := proto.Marshal(c)
		if err != nil {
			return err
		}
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(clustersPrefix, []byte(c.Id)), bytes)
	}
	return db.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
}

func getPrefix() []byte {
	prefix := make([]byte, 0, len(clustersPrefix)+len(separator))
	prefix = append(prefix, clustersPrefix...)
	prefix = append(prefix, separator...)
	return prefix
}
func init() {
	migrations.MustRegisterMigration(migration)
}
