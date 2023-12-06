package generic

import (
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
)

var (
	transactionPrefix = []byte("transactions")
)

// newTxnHelper returns a db wrapper that will increment txn counts
func newTxnHelper(db *rocksdb.RocksDB, objectType []byte, trackIndex bool) *txnHelper {
	wrapper := &txnHelper{
		db:         db,
		prefix:     append(transactionPrefix, objectType...),
		trackIndex: trackIndex,
	}

	return wrapper
}

// txnHelper overrides the Update function to increment txn counts
type txnHelper struct {
	db         *rocksdb.RocksDB
	trackIndex bool // if trackIndex is false then we don't need to mark the keys that need to be indexed
	prefix     []byte
}

// addKeysToIndex adds the keys not yet indexed to the DB
func (b *txnHelper) addKeysToIndex(batch *gorocksdb.WriteBatch, keys ...[]byte) {
	if !b.trackIndex {
		return
	}
	for _, k := range keys {
		batch.Put(dbhelper.GetBucketKey(b.prefix, k), []byte{0})
	}
}

// addStringKeysToIndex is a wrapper around addKeysToIndex but with a string slice instead of a []byte slice
func (b *txnHelper) addStringKeysToIndex(batch *gorocksdb.WriteBatch, keys ...string) {
	if !b.trackIndex {
		return
	}
	byteKeys := make([][]byte, 0, len(keys))
	for _, k := range keys {
		byteKeys = append(byteKeys, []byte(k))
	}
	b.addKeysToIndex(batch, byteKeys...)
}
