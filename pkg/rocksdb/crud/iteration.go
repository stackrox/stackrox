package generic

import (
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
)

// BucketKeyForEach ensures that the keys iterated over has the bucket prefix
func BucketKeyForEach(db *rocksdb.RocksDB, opts *gorocksdb.ReadOptions, keyPrefix []byte, stripPrefix bool, do func(k []byte) error) error {
	prefix := dbhelper.AppendSeparator(keyPrefix)
	return ForEachOverKeySet(db, opts, prefix, stripPrefix, do)
}

// DefaultBucketForEach runs BucketForEach with the default iterator options
func DefaultBucketForEach(db *rocksdb.RocksDB, keyPrefix []byte, stripPrefix bool, do func(k, v []byte) error) error {
	return BucketForEach(db, defaultIteratorOptions, keyPrefix, stripPrefix, do)
}

// BucketForEach iterates over a prefix with a key and value
func BucketForEach(db *rocksdb.RocksDB, opts *gorocksdb.ReadOptions, keyPrefix []byte, stripPrefix bool, do func(k, v []byte) error) error {
	prefix := dbhelper.AppendSeparator(keyPrefix)
	return ForEachItemWithPrefix(db, opts, prefix, stripPrefix, do)
}

// DefaultForEachOverKeySet invokes a callback for all keys with the given prefix.
func DefaultForEachOverKeySet(db *rocksdb.RocksDB, keyPrefix []byte, stripPrefix bool, do func(k []byte) error) error {
	return ForEachOverKeySet(db, defaultIteratorOptions, keyPrefix, stripPrefix, do)
}

// ForEachOverKeySet invokes a callback for all keys with the given prefix.
func ForEachOverKeySet(db *rocksdb.RocksDB, opts *gorocksdb.ReadOptions, keyPrefix []byte, stripPrefix bool, do func(k []byte) error) error {
	return ForEachItemWithPrefix(db, opts, keyPrefix, stripPrefix, func(k, v []byte) error {
		return do(k)
	})
}

// DefaultForEachItemWithPrefix invokes ForEachItemWithPrefix with the default read options
func DefaultForEachItemWithPrefix(db *rocksdb.RocksDB, keyPrefix []byte, stripPrefix bool, do func(k []byte, v []byte) error) error {
	return ForEachItemWithPrefix(db, defaultIteratorOptions, keyPrefix, stripPrefix, do)
}

// ForEachItemWithPrefix invokes a callbacks for all key/item pairs with the given prefix.
func ForEachItemWithPrefix(db *rocksdb.RocksDB, readOpts *gorocksdb.ReadOptions, keyPrefix []byte, stripPrefix bool, do func(k []byte, v []byte) error) error {
	it := db.NewIterator(readOpts)
	defer it.Close()

	for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
		k := it.Key().Data()
		if stripPrefix {
			k = dbhelper.StripPrefix(keyPrefix, k)
		}
		if err := do(k, it.Value().Data()); err != nil {
			return err
		}
	}
	return it.Err()
}
