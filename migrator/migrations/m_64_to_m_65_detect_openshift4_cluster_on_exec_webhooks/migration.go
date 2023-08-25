package m64to65

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	clustersPrefix = []byte("clusters")
	separator      = []byte("\x00")
)

var (
	migration = types.Migration{
		StartingSeqNum: 64,
		VersionAfter:   &storage.Version{SeqNum: 65},
		Run: func(databases *types.Databases) error {
			return migrateOpenShiftClusterType(databases.RocksDB)
		},
	}
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func migrateOpenShiftClusterType(db *gorocksdb.DB) error {
	clustersToMigrate := make(map[string]*storage.Cluster)
	readOpts := gorocksdb.NewDefaultReadOptions()
	it := db.NewIterator(readOpts)
	defer it.Close()

	prefix := getPrefix()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		cluster := &storage.Cluster{}
		if err := proto.Unmarshal(it.Value().Data(), cluster); err != nil {
			// If anything fails to unmarshal roll back the transaction and abort
			return errors.Wrapf(err, "Failed to unmarshal cluster data for key %s", it.Key().Data())
		}
		if cluster.GetType() != storage.ClusterType_OPENSHIFT_CLUSTER {
			continue
		}

		// Only migrate openshift 3 clusters to openshift 4 if they have admission controller events enabled
		// otherwise stay in openshift 3 compatibility mode.
		if cluster.GetAdmissionControllerEvents() {
			cluster.Type = storage.ClusterType_OPENSHIFT4_CLUSTER
			clustersToMigrate[string(it.Key().Data())] = cluster
		}
	}

	if len(clustersToMigrate) == 0 {
		return nil // nothing to do
	}
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()

	for k, c := range clustersToMigrate {
		bytes, err := proto.Marshal(c)
		if err != nil {
			return err
		}
		rocksWriteBatch.Put([]byte(k), bytes)
	}
	return db.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
}

func getPrefix() []byte {
	prefix := make([]byte, 0, len(clustersPrefix)+len(separator))
	prefix = append(prefix, clustersPrefix...)
	prefix = append(prefix, separator...)
	return prefix
}
