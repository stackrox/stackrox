package m27tom28

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

func getKeysWithPrefix(prefix []byte, db *badger.DB) ([][]byte, error) {
	prefixWithSep := prefixKey(prefix, nil)
	itOpts := &badger.IteratorOptions{
		PrefetchValues: false,
	}
	itOpts.Prefix = prefixWithSep

	var ret [][]byte
	err := db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(*itOpts)
		defer it.Close()
		for it.Seek(prefixWithSep); it.ValidForPrefix(prefixWithSep); it.Next() {
			item := it.Item()
			k := item.Key()

			copied := make([]byte, len(k))
			copy(copied, k)
			ret = append(ret, copied)
		}
		return nil
	})
	return ret, err
}

func readMapping(db *badger.DB, from []byte) (SortedKeys, error) {
	var tos SortedKeys
	err := db.View(func(txn *badger.Txn) error {
		// Read the top level object from the DB.
		item, err := txn.Get(getGraphKey(string(from)))
		if err == badger.ErrKeyNotFound {
			return nil
		} else if err != nil {
			return err
		}
		if err := item.Value(func(val []byte) error {
			tos, err = Unmarshal(val)
			return err
		}); err != nil {
			return errors.Wrapf(err, "unable to read mappings from key %s", from)
		}
		return nil
	})
	return tos, err
}

func writeMappings(batch *badger.WriteBatch, mappings map[string]SortedKeys) error {
	for from, tos := range mappings {
		if err := batch.Set(getGraphKey(from), tos.Marshal()); err != nil {
			return err
		}
	}
	return nil
}

func readProto(db *badger.DB, key []byte, msg proto.Message) (bool, error) {
	var exists bool
	err := db.View(func(txn *badger.Txn) error {
		// Read the top level object from the DB.
		item, err := txn.Get(key)
		if err == badger.ErrKeyNotFound {
			return nil
		} else if err != nil {
			return err
		}

		if err := item.Value(func(val []byte) error {
			if len(val) == 0 {
				return nil
			}
			exists = true
			return proto.Unmarshal(val, msg)
		}); err != nil {
			return errors.Wrapf(err, "unable to read key %s into type %T", string(key), msg)
		}
		return nil
	})
	return exists, err
}

func writeProto(batch *badger.WriteBatch, key []byte, msg proto.Message) error {
	toWrite, err := proto.Marshal(msg)
	if err != nil {
		return err
	}

	err = batch.Set(key, toWrite)
	if err != nil {
		return err
	}
	return nil
}
