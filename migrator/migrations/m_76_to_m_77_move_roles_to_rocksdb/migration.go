package m76to77

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/migrations/rocksdbmigration"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
	"go.etcd.io/bbolt"
)

var (
	migration = types.Migration{
		StartingSeqNum: 76,
		VersionAfter:   &storage.Version{SeqNum: 77},
		Run: func(databases *types.Databases) error {
			err := migrateRoles(databases.BoltDB, databases.RocksDB)
			if err != nil {
				return errors.Wrap(err, "moving roles from boltdb to rocksdb")
			}
			return nil
		},
	}
	rolesBucket = []byte("roles")
)

func migrateRoles(boltdb *bbolt.DB, rocksdb *gorocksdb.DB) error {
	// Collect roles which need migration.
	rolesToMigrate := make([]*storage.Role, 0)
	err := boltdb.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(rolesBucket)
		if bucket == nil {
			return errors.Errorf("bucket %s not found", rolesBucket)
		}
		return bucket.ForEach(func(k, v []byte) error {
			role := &storage.Role{}
			if err := proto.Unmarshal(v, role); err != nil {
				return errors.Wrapf(err, "failed to unmarshal role data for key %s", k)
			}
			rolesToMigrate = append(rolesToMigrate, role)
			return nil
		})
	})
	if err != nil {
		return errors.Wrap(err, "failed to read role data")
	}

	// Add roles to rocksdb database.
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	defer rocksWriteBatch.Destroy()
	for _, role := range rolesToMigrate {
		name := role.GetName()
		bytes, err := proto.Marshal(role)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal role data for name %q", name)
		}
		rocksWriteBatch.Put(rocksdbmigration.GetPrefixedKey(rolesBucket, []byte(name)), bytes)
	}
	err = rocksdb.Write(gorocksdb.NewDefaultWriteOptions(), rocksWriteBatch)
	if err != nil {
		return errors.Wrap(err, "failed to write to rocksdb")
	}

	// Delete roles bucket from boltdb database.
	err = boltdb.Update(func(tx *bbolt.Tx) error {
		err = tx.DeleteBucket(rolesBucket)
		return err
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete roles bucket from boltdb")
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
