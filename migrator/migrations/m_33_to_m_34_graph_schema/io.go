package m33tom34

import (
	"github.com/dgraph-io/badger"
	"github.com/pkg/errors"
)

func readMappings(db *badger.DB, prefix []byte) (map[string]SortedKeys, error) {
	graphPrefix := getFullPrefix(graphBucket)
	graphAndInputPrefix := getGraphKey(prefix)
	itOpts := &badger.IteratorOptions{
		PrefetchValues: true,
	}
	itOpts.Prefix = graphAndInputPrefix

	ret := make(map[string]SortedKeys)
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(*itOpts)
		defer it.Close()
		for it.Seek(graphAndInputPrefix); it.ValidForPrefix(graphAndInputPrefix); it.Next() {
			item := it.Item()
			key := item.Key()

			var tos SortedKeys
			var err error
			if err := item.Value(func(val []byte) error {
				tos, err = Unmarshal(val)
				return err
			}); err != nil {
				return errors.Wrapf(err, "unable to read mappings from key %s", key)
			}
			ret[string(key[len(graphPrefix):])] = tos
		}
		return nil
	})
	return ret, err
}

func writeMappings(batch *badger.WriteBatch, mappings map[string]SortedKeys) error {
	for from, tos := range mappings {
		if err := batch.Set(getGraphKey([]byte(from)), tos.Marshal()); err != nil {
			return err
		}
	}
	return nil
}
