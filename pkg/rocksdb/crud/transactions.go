package generic

import (
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/logging"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
)

var (
	transactionPrefix = []byte("transactions")

	log = logging.LoggerForModule()
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

// AckKeysIndexed acknowledges that keys were indexed
func (b *txnHelper) AckKeysIndexed(keys ...string) error {
	if !b.trackIndex {
		log.Errorf("UNEXPECTED: acking keys indexed for prefix %s despite having trackIndex=false. TrackIndex should probably be set to true", b.prefix)
		return nil
	}
	if err := b.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer b.db.DecRocksDBInProgressOps()

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
	if !b.trackIndex {
		return nil, nil
	}
	if err := b.db.IncRocksDBInProgressOps(); err != nil {
		return nil, err
	}
	defer b.db.DecRocksDBInProgressOps()

	var keys []string
	err := BucketKeyForEach(b.db, defaultIteratorOptions, b.prefix, true, func(k []byte) error {
		keys = append(keys, string(k))
		return nil
	})
	return keys, errors.Wrap(err, "getting keys to index")
}
