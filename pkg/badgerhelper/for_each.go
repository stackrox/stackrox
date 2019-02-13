package badgerhelper

import "github.com/dgraph-io/badger"

// ForEachOptions controls the behavior of a `ForEachWithPrefix` call.
type ForEachOptions struct {
	StripKeyPrefix  bool
	IteratorOptions *badger.IteratorOptions
}

// ForEachWithPrefix invokes a callback for all key/value pairs with the given prefix.
func ForEachWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
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

		err := item.Value(func(val []byte) error {
			return do(k, val)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
