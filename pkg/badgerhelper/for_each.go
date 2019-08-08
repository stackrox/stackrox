package badgerhelper

import "github.com/dgraph-io/badger"

// ForEachOptions controls the behavior of a `ForEach[Item]WithPrefix` call.
type ForEachOptions struct {
	StripKeyPrefix  bool
	IteratorOptions *badger.IteratorOptions
}

// ForEachItemWithPrefix invokes a callbacks for all key/item pairs with the given prefix.
func ForEachItemWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte, item *badger.Item) error) error {
	itOpts := badger.DefaultIteratorOptions
	if opts.IteratorOptions != nil {
		itOpts = *opts.IteratorOptions
	}
	itOpts.Prefix = keyPrefix

	it := txn.NewIterator(itOpts)
	defer it.Close()
	for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
		item := it.Item()
		k := item.Key()
		if opts.StripKeyPrefix {
			k = StripPrefix(keyPrefix, k)
		}

		if err := do(k, item); err != nil {
			return err
		}
	}
	return nil
}

// StripBucket removes a bucket prefix and the separator from the val
func StripBucket(prefix []byte, val []byte) []byte {
	bucket := GetBucketKey(prefix, nil)
	return StripPrefix(bucket, val)
}

// StripPrefix removes a prefix from the val
func StripPrefix(prefix []byte, val []byte) []byte {
	if len(val) >= len(prefix) {
		return val[len(prefix):]
	}
	return val
}

// BucketForEach ensures that the prefix iterated over has the bucket prefix
func BucketForEach(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
	keyPrefix = append(keyPrefix, separator...)
	return ForEachWithPrefix(txn, keyPrefix, opts, do)
}

// BucketKeyForEach ensures that the keys iterated over has the bucket prefix
func BucketKeyForEach(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte) error) error {
	keyPrefix = append(keyPrefix, separator...)
	return ForEachOverKeySet(txn, keyPrefix, opts, do)
}

// ForEachWithPrefix invokes a callback for all key/value pairs with the given prefix.
func ForEachWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
	closure := func(k []byte, item *badger.Item) error {
		return item.Value(func(v []byte) error {
			return do(k, v)
		})
	}
	return ForEachItemWithPrefix(txn, keyPrefix, opts, closure)
}

// ForEachOverKeySet invokes a callback for all keys with the given prefix.
func ForEachOverKeySet(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte) error) error {
	closure := func(k []byte, _ *badger.Item) error {
		return do(k)
	}
	if opts.IteratorOptions == nil {
		opts.IteratorOptions = &badger.DefaultIteratorOptions
	}
	opts.IteratorOptions.PrefetchValues = false
	return ForEachItemWithPrefix(txn, keyPrefix, opts, closure)
}
