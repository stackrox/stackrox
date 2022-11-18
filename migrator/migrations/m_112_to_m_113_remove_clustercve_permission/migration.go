package m112tom113

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

const (
	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: 112,
		VersionAfter:   &storage.Version{SeqNum: 113},
		Run: func(databases *types.Databases) error {
			err := cleanupPermissionSets(databases.RocksDB)
			if err != nil {
				return errors.Wrap(err, "updating PermissionSet schema")
			}
			return nil
		},
	}

	permissionSetPrefix = []byte("permission_sets")

	clusterCVEResourceName = "ClusterCVE"

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func cleanupPermissionSets(db *gorocksdb.DB) error {
	it := db.NewIterator(readOpts)
	defer it.Close()
	wb := gorocksdb.NewWriteBatch()
	for it.Seek(permissionSetPrefix); it.ValidForPrefix(permissionSetPrefix); it.Next() {
		ps := &storage.PermissionSet{}
		if err := ps.Unmarshal(it.Value().Data()); err != nil {
			return errors.Wrap(err, "unable to unmarshal permission set")
		}
		if _, ok := ps.GetResourceToAccess()[clusterCVEResourceName]; ok {
			delete(ps.ResourceToAccess, clusterCVEResourceName)
			data, err := ps.Marshal()
			if err != nil {
				return errors.Wrap(err, "unable to marshal permission set")
			}
			wb.Put(it.Key().Copy(), data)
			if wb.Count() == batchSize {
				if err := db.Write(writeOpts, wb); err != nil {
					return errors.Wrap(err, "writing to RocksDB")
				}
				wb.Clear()
			}
		}
	}
	if wb.Count() != 0 {
		if err := db.Write(writeOpts, wb); err != nil {
			return errors.Wrap(err, "writing final batch to RocksDB")
		}
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
