package generic

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/tecbot/gorocksdb"
)

var (
	transactionPrefix = []byte("transactions")
)

// newTxnHelper returns a db wrapper that will increment txn counts
func newTxnHelper(db *gorocksdb.DB, objectType []byte) *txnHelper {
	wrapper := &txnHelper{
		db:     db,
		prefix: append(transactionPrefix, objectType...),
	}

	return wrapper
}

// txnHelper overrides the Update function to increment txn counts
type txnHelper struct {
	db *gorocksdb.DB

	prefix []byte
}

// addKeysToIndex adds the keys not yet indexed to the DB
func (b *txnHelper) addKeysToIndex(batch *gorocksdb.WriteBatch, keys ...[]byte) {
	for _, k := range keys {
		batch.Put(dbhelper.GetBucketKey(b.prefix, k), []byte{0})
	}
}

// AddStringKeysToIndex is a wrapper around addKeysToIndex but with a string slice instead of a []byte slice
func (b *txnHelper) AddStringKeysToIndex(batch *gorocksdb.WriteBatch, keys ...string) {
	byteKeys := make([][]byte, 0, len(keys))
	for _, k := range keys {
		byteKeys = append(byteKeys, []byte(k))
	}
	b.addKeysToIndex(batch, byteKeys...)
}

// AckKeysIndexed acknowledges that keys were indexed
func (b *txnHelper) AckKeysIndexed(keys ...string) error {
	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	for _, k := range keys {
		batch.Delete(dbhelper.GetBucketKey(b.prefix, []byte(k)))
	}

	if err := b.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrap(err, "acking indexed keys")
	}
	return nil
}

// GetKeysToIndex retrieves the number of keys to index
func (b *txnHelper) GetKeysToIndex() ([]string, error) {
	var keys []string
	err := BucketKeyForEach(b.db, defaultIteratorOptions, b.prefix, true, func(k []byte) error {
		keys = append(keys, string(k))
		return nil
	})
	return keys, errors.Wrap(err, "getting keys to index")
}
