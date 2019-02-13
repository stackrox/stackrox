package badgerhelper

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/batcher"
	"github.com/stackrox/rox/pkg/bolthelper"
)

// PutAll inserts the given key/value pairs into the DB. Its main use case is to reduce the time the write lock is held
// for bulk upserts, by moving serialization outside of the transaction.
func PutAll(txn *badger.Txn, kvs ...bolthelper.KV) error {
	for _, kv := range kvs {
		if err := txn.Set(kv.Key, kv.Value); err != nil {
			return err
		}
	}
	return nil
}

// PutAllBatched inserts the given key/value pairs into the DB in batches of size batchSize. The first return value
// indicates the number of key/value pairs successfully written; if an error is returned, it will be less than len(kvs),
// and equal to len(kvs) in the non-error case.
func PutAllBatched(db *badger.DB, kvs []bolthelper.KV, batchSize int) (int, error) {
	b := batcher.New(len(kvs), batchSize)

	for start, end, valid := b.Next(); valid; start, end, valid = b.Next() {
		if err := db.Update(func(txn *badger.Txn) error { return PutAll(txn, kvs[start:end]...) }); err != nil {
			return start, err
		}
	}
	return len(kvs), nil
}
