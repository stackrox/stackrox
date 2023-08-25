package m95tom96

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
	alertBucket = []byte("alerts")

	migration = types.Migration{
		StartingSeqNum: 95,
		VersionAfter:   &storage.Version{SeqNum: 96},
		Run:            copyAlertScopingInformationToRoot,
	}

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func init() {
	migrations.MustRegisterMigration(migration)
}

func copyAlertScopingInformationToRoot(db *types.Databases) error {
	it := db.RocksDB.NewIterator(readOpts)
	defer it.Close()

	prefix := rocksdbmigration.GetBucketPrefix(alertBucket)

	wb := gorocksdb.NewWriteBatch()
	defer wb.Destroy()
	for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
		key := it.Key().Copy()

		var alert storage.Alert
		err := proto.Unmarshal(it.Value().Data(), &alert)
		if err != nil {
			return err
		}

		alertEntity := alert.GetEntity()
		switch alertEntity.(type) {
		case *storage.Alert_Deployment_:
			entity := alert.GetDeployment()
			alert.ClusterId = entity.ClusterId
			alert.ClusterName = entity.ClusterName
			alert.Namespace = entity.Namespace
			alert.NamespaceId = entity.NamespaceId
		case *storage.Alert_Resource_:
			entity := alert.GetResource()
			alert.ClusterId = entity.ClusterId
			alert.ClusterName = entity.ClusterName
			alert.Namespace = entity.Namespace
			alert.NamespaceId = entity.NamespaceId
		case *storage.Alert_Image:
			alert.ClusterId = ""
			alert.ClusterName = ""
			alert.Namespace = ""
			alert.NamespaceId = ""
		}

		data, err := proto.Marshal(&alert)
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
