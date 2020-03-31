// +build rocksdb

package generic

import (
	"github.com/stackrox/rox/pkg/dbhelper"
	"github.com/tecbot/gorocksdb"
)

// BucketKeyForEach ensures that the keys iterated over has the bucket prefix
func BucketKeyForEach(db *gorocksdb.DB, keyPrefix []byte, stripPrefix bool, do func(k []byte) error) error {
	prefix := dbhelper.AppendSeparator(keyPrefix)
	return ForEachOverKeySet(db, prefix, stripPrefix, do)
}

// BucketForEach iterates over a prefix with a key and value
func BucketForEach(db *gorocksdb.DB, keyPrefix []byte, stripPrefix bool, do func(k, v []byte) error) error {
	prefix := dbhelper.AppendSeparator(keyPrefix)
	return ForEachItemWithPrefix(db, prefix, stripPrefix, do)
}

// ForEachOverKeySet invokes a callback for all keys with the given prefix.
func ForEachOverKeySet(db *gorocksdb.DB, keyPrefix []byte, stripPrefix bool, do func(k []byte) error) error {
	return ForEachItemWithPrefix(db, keyPrefix, stripPrefix, func(k, v []byte) error {
		return do(k)
	})
}

// ForEachItemWithPrefix invokes a callbacks for all key/item pairs with the given prefix.
func ForEachItemWithPrefix(db *gorocksdb.DB, keyPrefix []byte, stripPrefix bool, do func(k []byte, v []byte) error) error {
	it := db.NewIterator(defaultIteratorOptions)
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
	return nil
}
