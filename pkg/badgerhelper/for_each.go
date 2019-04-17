package badgerhelper

import "github.com/dgraph-io/badger"

// ForEachOptions controls the behavior of a `ForEachWithPrefix` call.
type ForEachOptions struct {
	StripKeyPrefix  bool
	IteratorOptions *badger.IteratorOptions
}

func forEachWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte, item *badger.Item) error) error {
	itOpts := badger.DefaultIteratorOptions
	if opts.IteratorOptions != nil {
		itOpts = *opts.IteratorOptions
	}

	it := txn.NewIterator(itOpts)
	defer it.Close()
	for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
		item := it.Item()
		k := item.Key()
		if opts.StripKeyPrefix {
			k = k[len(keyPrefix):]
		}

		if err := do(k, item); err != nil {
			return err
		}
	}
	return nil
}

// ForEachWithPrefix invokes a callback for all key/value pairs with the given prefix.
func ForEachWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
	closure := func(k []byte, item *badger.Item) error {
		return item.Value(func(v []byte) error {
			return do(k, v)
		})
	}
	return forEachWithPrefix(txn, keyPrefix, opts, closure)
}

// ForEachOverKeySet invokes a callback for all keys with the given prefix.
func ForEachOverKeySet(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte) error) error {
	closure := func(k []byte, _ *badger.Item) error {
		return do(k)
	}
	return forEachWithPrefix(txn, keyPrefix, opts, closure)
}
