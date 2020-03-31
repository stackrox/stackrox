package badgerhelper

import (
	"github.com/dgraph-io/badger"
	"github.com/stackrox/rox/pkg/dbhelper"
)

var (
	transactionPrefix = []byte("transactions")
)

// NewTxnHelper returns a db wrapper that will increment txn counts
func NewTxnHelper(db *badger.DB, objectType []byte) (*TxnHelper, error) {
	wrapper := &TxnHelper{
		db:     db,
		prefix: append(transactionPrefix, objectType...),
	}

	return wrapper, nil
}

// TxnHelper overrides the Update function to increment txn counts
type TxnHelper struct {
	db *badger.DB

	prefix []byte
}

// TxWrapper wraps a txn to expose the Set interface
type TxWrapper interface {
	Set(k, v []byte) error
}

// AddKeysToIndex adds the keys not yet indexed to the DB
func (b *TxnHelper) AddKeysToIndex(tx TxWrapper, keys ...[]byte) error {
	for _, k := range keys {
		if err := tx.Set(dbhelper.GetBucketKey(b.prefix, k), []byte{0}); err != nil {
			return err
		}
	}
	return nil
}

// AddStringKeysToIndex is a wrapper around AddKeysToIndex but with a string slice instead of a []byte slice
func (b *TxnHelper) AddStringKeysToIndex(tx TxWrapper, keys ...string) error {
	byteKeys := make([][]byte, 0, len(keys))
	for _, k := range keys {
		byteKeys = append(byteKeys, []byte(k))
	}
	return b.AddKeysToIndex(tx, byteKeys...)
}

// AckKeysIndexed acknowledges that keys were indexed
func (b *TxnHelper) AckKeysIndexed(keys ...string) error {
	batch := b.db.NewWriteBatch()
	defer batch.Cancel()

	for _, k := range keys {
		if err := batch.Delete(dbhelper.GetBucketKey(b.prefix, []byte(k))); err != nil {
			return err
		}
	}
	return batch.Flush()
}

// GetKeysToIndex retrieves the number of keys to index
func (b *TxnHelper) GetKeysToIndex() ([]string, error) {
	var keys []string
	err := b.db.View(func(tx *badger.Txn) error {
		return BucketKeyForEach(tx, b.prefix, ForEachOptions{
			StripKeyPrefix: true,
		}, func(k []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
}
