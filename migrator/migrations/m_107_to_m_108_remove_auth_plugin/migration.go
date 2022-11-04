package m107tom108

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
)

const (
	batchSize = 500
)

var (
	migration = types.Migration{
		StartingSeqNum: 107,
		VersionAfter:   &storage.Version{SeqNum: 108},
		Run: func(databases *types.Databases) error {
			err := migratePS(databases.RocksDB)
			if err != nil {
				return errors.Wrap(err, "updating PermissionSet schema")
			}
			return deleteAuthPluginBucket(databases.BoltDB)
		},
	}

	psPrefix               = []byte("permission_sets")
	authPluginBucket       = []byte("authzPlugins")
	authPluginResourceName = "AuthPlugin"

	readOpts  = gorocksdb.NewDefaultReadOptions()
	writeOpts = gorocksdb.NewDefaultWriteOptions()
)

func migratePS(db *gorocksdb.DB) error {

	it := db.NewIterator(readOpts)
	defer it.Close()
	wb := gorocksdb.NewWriteBatch()
	for it.Seek(psPrefix); it.ValidForPrefix(psPrefix); it.Next() {
		ps := &storage.PermissionSet{}
		if err := proto.Unmarshal(it.Value().Data(), ps); err != nil {
			return errors.Wrap(err, "unable to unmarshal permission set")
		}
		if _, ok := ps.ResourceToAccess[authPluginResourceName]; ok {
			delete(ps.ResourceToAccess, authPluginResourceName)
			data, err := proto.Marshal(ps)
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

func deleteAuthPluginBucket(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		if b := tx.Bucket(authPluginBucket); b == nil {
			// nothing to be done if the bucket doesn't exist
			return nil
		}
		return tx.DeleteBucket(authPluginBucket)
	})
}

func init() {
	migrations.MustRegisterMigration(migration)
}
