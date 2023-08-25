package m94tom95

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	clusterHealthStatusBucket = []byte("cluster_health_status")

	migration = types.Migration{
		StartingSeqNum: 94,
		VersionAfter:   &storage.Version{SeqNum: 95},
		Run:            addIDToClusterHealthStatus,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func addIDToClusterHealthStatus(db *types.Databases) error {
	it := db.RocksDB.NewIterator(readOpts)
	defer it.Close()

	prefix := rocksdbmigration.GetBucketPrefix(clusterHealthStatusBucket)

	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key().Copy()

		var healthStatus storage.ClusterHealthStatus
		err := proto.Unmarshal(it.Value().Data(), &healthStatus)
		if err != nil {
			return err
		}

		healthStatus.Id = string(rocksdbmigration.GetPrefixedKey(prefix, key))

		data, err := proto.Marshal(&healthStatus)
		if err != nil {
			return err
		}
		wb.Put(key, data)

		if wb.Count() == batchSize {
			if err := db.RocksDB.Write(writeOpts, wb); err != nil {
				return errors.Wrap(err, "writing to RocksDB")
			}
			wb.Clear()
		}
	}

	if wb.Count() != 0 {
		if err := db.RocksDB.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}
