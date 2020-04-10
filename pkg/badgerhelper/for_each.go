package badgerhelper

import (
	"github.com/stackrox/rox/pkg/dbhelper"

	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/assert"
	"github.com/stackrox/rox/pkg/utils"
)

// ForEachOptions controls the behavior of a `ForEach[Item]WithPrefix` call.
type ForEachOptions struct {
	StripKeyPrefix  bool
	IteratorOptions *badger.IteratorOptions
}

// DefaultIteratorOptions defines the default iterator options for Badger
// These should be used instead of badger.DefaultIteratorOptions
func DefaultIteratorOptions() *badger.IteratorOptions {
	return &badger.IteratorOptions{
		PrefetchValues: false,
	}
}

// ForEachItemWithPrefix invokes a callbacks for all key/item pairs with the given prefix.
func ForEachItemWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte, item *badger.Item) error) error {
	panicked := true
	defer func() {
		if r := recover(); r != nil || panicked {
			utils.Should(errors.Errorf("panic in iteration: %+v", r))
		}
	}()

	itOpts := DefaultIteratorOptions()
	if opts.IteratorOptions != nil {
		itOpts = opts.IteratorOptions
	}
	itOpts.Prefix = keyPrefix

	it := txn.NewIterator(*itOpts)
	defer it.Close()
	for it.Seek(keyPrefix); it.ValidForPrefix(keyPrefix); it.Next() {
		item := it.Item()
		k := item.Key()
		if opts.StripKeyPrefix {
			k = dbhelper.StripPrefix(keyPrefix, k)
		}

		if err := do(k, item); err != nil {
			return err
		}
	}
	panicked = false
	return nil
}

// BucketForEach ensures that the prefix iterated over has the bucket prefix
func BucketForEach(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
	bucketPrefix := dbhelper.AppendSeparator(keyPrefix)
	return ForEachWithPrefix(txn, bucketPrefix, opts, do)
}

// BucketKeyForEach ensures that the keys iterated over has the bucket prefix
func BucketKeyForEach(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k []byte) error) error {
	bucketPrefix := dbhelper.AppendSeparator(keyPrefix)
	return ForEachOverKeySet(txn, bucketPrefix, opts, do)
}

// ForEachWithPrefix invokes a callback for all key/value pairs with the given prefix.
func ForEachWithPrefix(txn *badger.Txn, keyPrefix []byte, opts ForEachOptions, do func(k, v []byte) error) error {
	closure := func(k []byte, item *badger.Item) error {
		return item.Value(func(v []byte) error {
			if len(v) == 0 {
				assert.Panicf("%s has value of length 0", k)
				return nil
			}
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
	itOpts := DefaultIteratorOptions()
	if opts.IteratorOptions == nil {
		opts.IteratorOptions = itOpts
	}
	opts.IteratorOptions.PrefetchValues = false
	return ForEachItemWithPrefix(txn, keyPrefix, opts, closure)
}
