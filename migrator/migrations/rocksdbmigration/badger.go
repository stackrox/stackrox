package rocksdbmigration

import (
	"bytes"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/migrator/types"
	"github.com/tecbot/gorocksdb"
)

var (
	versionKey = []byte("version")
)

func migrateBadger(databases *types.Databases) error {
	rocksWriteBatch := gorocksdb.NewWriteBatch()
	writeOptions := gorocksdb.NewDefaultWriteOptions()
	defer writeOptions.Destroy()

	count := 0
	err := databases.BadgerDB.View(func(txn *badger.Txn) error {
		itOpts := badger.DefaultIteratorOptions
		it := txn.NewIterator(itOpts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			// Purposefully do not write out the version key as this will be used to distinguish when
			// the migration has been successfully completed
			if bytes.HasPrefix(it.Item().Key(), versionKey) {
				continue
			}

			value, err := it.Item().ValueCopy(nil)
			if err != nil {
				return errors.Wrap(err, "value copying during migration")
			}
			if len(value) == 0 {
				continue
			}
			rocksWriteBatch.Put(it.Item().KeyCopy(nil), value)
			count++
			if count%batchSize == 0 {
				if err := databases.RocksDB.Write(writeOptions, rocksWriteBatch); err != nil {
					return errors.Wrap(err, "writing batch to rocksdb")
				}
				rocksWriteBatch.Clear()
			}
		}
		// Write out the remaining in the batch
		if err := databases.RocksDB.Write(writeOptions, rocksWriteBatch); err != nil {
			return errors.Wrap(err, "writing batch to rocksdb")
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "migrating badgerDB to RocksDB")
	}
	return nil
}
