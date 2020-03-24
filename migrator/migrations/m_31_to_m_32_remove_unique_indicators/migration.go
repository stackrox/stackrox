package m31to32

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/process/id"
	bolt "go.etcd.io/bbolt"
)

var (
	uniqueProcessBucket = []byte("process_indicators_unique\x00")
	oldProcessBucket    = []byte("process_indicators\x00")
	newProcessBucket    = []byte("process_indicators2\x00")
	migration           = types.Migration{
		StartingSeqNum: 31,
		VersionAfter:   storage.Version{SeqNum: 32},
		Run:            removeUniqueProcessPrefix,
	}
)

func removePrefix(db *badger.DB, prefix []byte) error {
	removeBatch := db.NewWriteBatch()
	defer removeBatch.Cancel()

	err := db.View(func(tx *badger.Txn) error {
		itOpts := badger.DefaultIteratorOptions
		itOpts.Prefix = prefix
		it := tx.NewIterator(itOpts)
		defer it.Close()
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if err := removeBatch.Delete(it.Item().KeyCopy(nil)); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return removeBatch.Flush()
}

func removeUniqueProcessPrefix(_ *bolt.DB, badgerDB *badger.DB) error {
	if err := removePrefix(badgerDB, uniqueProcessBucket); err != nil {
		return err
	}

	// Migrate all from process_indicators -> process_indicators2
	newBatch := badgerDB.NewWriteBatch()
	defer newBatch.Cancel()

	count := 0
	err := badgerDB.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = oldProcessBucket
		it := tx.NewIterator(opts)
		defer it.Close()
		for it.Seek(oldProcessBucket); it.ValidForPrefix(oldProcessBucket); it.Next() {
			var indicator storage.ProcessIndicator
			err := it.Item().Value(func(v []byte) error {
				return proto.Unmarshal(v, &indicator)
			})
			if err != nil {
				return err
			}

			id.SetIndicatorID(&indicator)
			value, err := proto.Marshal(&indicator)
			if err != nil {
				return err
			}

			newKey := append([]byte{}, newProcessBucket...)
			newKey = append(newKey, []byte(indicator.GetId())...)

			if err := newBatch.Set(newKey, value); err != nil {
				return err
			}
			count++
			if count%10000 == 0 {
				log.WriteToStderrf("Wrote %d indicators to batch", count)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := newBatch.Flush(); err != nil {
		return err
	}

	return removePrefix(badgerDB, oldProcessBucket)
}

func init() {
	migrations.MustRegisterMigration(migration)
}
