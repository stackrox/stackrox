package m31to32

import (
	"github.com/dgraph-io/badger"
	bolt "github.com/etcd-io/bbolt"
	"github.com/gogo/protobuf/proto"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/log"
	"github.com/stackrox/rox/migrator/migrations"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/process/id"
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

func removeUniqueProcessPrefix(_ *bolt.DB, badgerDB *badger.DB) error {
	// Generally, we try to avoid drop prefix because if multiple run at once, there have been some issues filed.
	// In this case, it'll be the only thing run so it should be safe
	if err := badgerDB.DropPrefix(uniqueProcessBucket); err != nil {
		return err
	}

	// Migrate all from process_indicators -> process_indicators2
	newBatch := badgerDB.NewWriteBatch()
	defer newBatch.Cancel()

	count := 0
	err := badgerDB.View(func(tx *badger.Txn) error {
		it := tx.NewIterator(badger.DefaultIteratorOptions)
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

	if err := badgerDB.DropPrefix(oldProcessBucket); err != nil {
		return err
	}
	return nil
}

func init() {
	migrations.MustRegisterMigration(migration)
}
