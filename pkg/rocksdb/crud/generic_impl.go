package generic

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/dbhelper"
	"github.com/stackrox/stackrox/pkg/rocksdb"
	"github.com/tecbot/gorocksdb"
)

var (
	defaultReadOptions = DefaultReadOptions()

	defaultWriteOptions = DefaultWriteOptions()

	defaultIteratorOptions = DefaultIteratorOptions()
)

type crudImpl struct {
	*txnHelper
	db *rocksdb.RocksDB

	prefix          []byte
	keyFunc         KeyFunc
	alloc           AllocFunc
	deserializeFunc Deserializer
}

func (c *crudImpl) getPrefixedKey(id string) []byte {
	return dbhelper.GetBucketKey(c.prefix, []byte(id))
}

func (c *crudImpl) getPrefixedKeyBytes(id []byte) []byte {
	return dbhelper.GetBucketKey(c.prefix, id)
}

func (c *crudImpl) Count() (int, error) {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return 0, err
	}
	defer c.db.DecRocksDBInProgressOps()

	var count int
	err := ForEachOverKeySet(c.db, defaultIteratorOptions, c.prefix, false, func(k []byte) error {
		count++
		return nil
	})
	return count, errors.Wrap(err, "getting count of objects in DB")
}

func (c *crudImpl) Get(id string) (proto.Message, bool, error) {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return nil, false, err
	}
	defer c.db.DecRocksDBInProgressOps()

	key := c.getPrefixedKey(id)
	slice, err := c.db.Get(defaultReadOptions, key)
	if err != nil {
		return nil, false, errors.Wrapf(err, "getting key %s", key)
	}
	defer slice.Free()
	if !slice.Exists() {
		return nil, false, nil
	}
	msg, err := c.deserializeFunc(slice.Data())
	if err != nil {
		return nil, false, errors.Wrapf(err, "deserializing object with key %s", key)
	}
	return msg, true, nil
}

func (c *crudImpl) Exists(id string) (exists bool, err error) {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return false, err
	}
	defer c.db.DecRocksDBInProgressOps()

	slice, err := c.db.Get(defaultReadOptions, c.getPrefixedKey(id))
	if err != nil {
		return false, errors.Wrapf(err, "getting id %s", id)
	}
	defer slice.Free()
	return slice.Exists(), nil
}

func (c *crudImpl) GetMany(ids []string) (msgs []proto.Message, missingIndices []int, err error) {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return nil, nil, err
	}
	defer c.db.DecRocksDBInProgressOps()

	keys := make([][]byte, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, c.getPrefixedKey(id))
	}

	slices, err := c.db.MultiGet(defaultReadOptions, keys...)
	if err != nil {
		return nil, nil, errors.Wrap(err, "running multiget")
	}
	for idx, slice := range slices {
		if !slice.Exists() {
			missingIndices = append(missingIndices, idx)
			slice.Free()
			continue
		}
		msg, err := c.deserializeFunc(slice.Data())
		if err != nil {
			return nil, nil, errors.Wrap(err, "deserializing object")
		}
		slice.Free()
		msgs = append(msgs, msg)
	}
	return msgs, missingIndices, nil
}

func (c *crudImpl) addToWriteBatch(batch *gorocksdb.WriteBatch, msg proto.Message) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshaling message")
	}
	nonPrefixedID := c.keyFunc(msg)
	return c.addKVToWriteBatch(batch, nonPrefixedID, data)
}

func (c *crudImpl) addKVToWriteBatch(batch *gorocksdb.WriteBatch, id []byte, data []byte) error {
	c.addKeysToIndex(batch, id)
	batch.Put(c.getPrefixedKeyBytes(id), data)
	return nil
}

func (c *crudImpl) Upsert(msg proto.Message) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	if err := c.addToWriteBatch(batch, msg); err != nil {
		return errors.Wrap(err, "adding to write batch")
	}

	if err := c.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrap(err, "writing to DB")
	}
	return nil
}

func (c *crudImpl) UpsertMany(msgs []proto.Message) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	for _, msg := range msgs {
		if err := c.addToWriteBatch(batch, msg); err != nil {
			return errors.Wrap(err, "adding to write batch")
		}
	}

	if err := c.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrap(err, "writing batch")
	}
	return nil
}

func (c *crudImpl) UpsertWithID(id string, msg proto.Message) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	data, err := proto.Marshal(msg)
	if err != nil {
		return errors.Wrap(err, "marshaling message")
	}

	if err := c.addKVToWriteBatch(batch, []byte(id), data); err != nil {
		return errors.Wrap(err, "adding to write batch")
	}

	if err := c.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrap(err, "writing to DB")
	}
	return nil
}

func (c *crudImpl) UpsertManyWithIDs(ids []string, msgs []proto.Message) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	if len(ids) != len(msgs) {
		return errors.Errorf("%s: length(ids) %d does not match len(msgs) %d", c.prefix, len(ids), len(msgs))
	}

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	for i, msg := range msgs {
		data, err := proto.Marshal(msg)
		if err != nil {
			return errors.Wrap(err, "marshaling message")
		}

		if err := c.addKVToWriteBatch(batch, []byte(ids[i]), data); err != nil {
			return errors.Wrap(err, "adding to write batch")
		}
	}

	if err := c.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrap(err, "writing batch")
	}
	return nil
}

func (c *crudImpl) Delete(id string) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	// Include keys to index within this transaction to avoid creating a new txn
	c.addStringKeysToIndex(batch, id)
	batch.Delete(c.getPrefixedKey(id))

	if err := c.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrapf(err, "deleting id %s", id)
	}
	return nil
}

func (c *crudImpl) DeleteMany(ids []string) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	batch := gorocksdb.NewWriteBatch()
	defer batch.Destroy()

	for _, id := range ids {
		c.addStringKeysToIndex(batch, id)
		batch.Delete(c.getPrefixedKey(id))
	}

	if err := c.db.Write(defaultWriteOptions, batch); err != nil {
		return errors.Wrap(err, "running delete many")
	}
	return nil
}

func (c *crudImpl) GetKeys() ([]string, error) {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return nil, err
	}
	defer c.db.DecRocksDBInProgressOps()

	var keys []string
	err := BucketKeyForEach(c.db, defaultIteratorOptions, c.prefix, true, func(k []byte) error {
		keys = append(keys, string(k))
		return nil
	})
	return keys, errors.Wrap(err, "getting keys")
}

func (c *crudImpl) Walk(fn func(msg proto.Message) error) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	return BucketForEach(c.db, defaultIteratorOptions, c.prefix, false, func(k, v []byte) error {
		msg, err := c.deserializeFunc(v)
		if err != nil {
			return err
		}
		if err := fn(msg); err != nil {
			return err
		}
		return nil
	})
}

func (c *crudImpl) WalkAllWithID(fn func(id []byte, msg proto.Message) error) error {
	if err := c.db.IncRocksDBInProgressOps(); err != nil {
		return err
	}
	defer c.db.DecRocksDBInProgressOps()

	return BucketForEach(c.db, defaultIteratorOptions, c.prefix, true, func(k, v []byte) error {
		msg, err := c.deserializeFunc(v)
		if err != nil {
			return errors.Wrap(err, "deserializing object")
		}
		if err := fn(k, msg); err != nil {
			return errors.Wrap(err, "applying closure")
		}
		return nil
	})
}
