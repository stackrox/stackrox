package generic

import (
	"github.com/dgraph-io/badger"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/badgerhelper"
	"github.com/stackrox/rox/pkg/logging"
)

type partialConvert func(msg proto.Message) proto.Message

type crudImpl struct {
	txnHelper *badgerhelper.TxnHelper
	db        *badger.DB

	prefix          []byte
	prefixString    string
	keyFunc         keyFunc
	alloc           allocFunc
	deserializeFunc Deserializer

	hasPartial             bool
	partialPrefix          []byte
	partialConverter       partialConvert
	partialAlloc           allocFunc
	partialDeserializeFunc Deserializer
}

var log = logging.LoggerForModule()

func (c *crudImpl) Count() (int, error) {
	var count int
	err := c.db.View(func(tx *badger.Txn) error {
		var err error
		count, err = badgerhelper.BucketKeyCount(tx, c.prefix)
		return err
	})
	return count, errors.Wrapf(err, "error getting count in %s", c.prefixString)
}

func (c *crudImpl) Create(msg proto.Message) error {
	return c.Upsert(msg)
}

func (c *crudImpl) CreateBatch(msg []proto.Message) error {
	return c.UpsertBatch(msg)
}

func (c *crudImpl) getKey(id string) []byte {
	return badgerhelper.GetBucketKey(c.prefix, []byte(id))
}

func (c *crudImpl) getPartialKey(id string) []byte {
	return badgerhelper.GetBucketKey(c.partialPrefix, []byte(id))
}

func read(tx *badger.Txn, prefix []byte, deserializer Deserializer, id string) (proto.Message, error) {
	var msg proto.Message
	key := badgerhelper.GetBucketKey(prefix, []byte(id))

	item, err := tx.Get(key)
	if err != nil {
		return nil, err
	}
	err = item.Value(func(v []byte) error {
		msg, err = deserializer(v)
		return err
	})
	return msg, err
}

func (c *crudImpl) read(id string, prefix []byte, deserializer Deserializer) (proto.Message, bool, error) {
	var (
		msg proto.Message
		err error
	)
	err = c.db.View(func(tx *badger.Txn) error {
		msg, err = read(tx, prefix, deserializer, id)
		return err
	})
	if err == badger.ErrKeyNotFound {
		return nil, false, nil
	}

	return msg, true, err
}

func (c *crudImpl) Read(id string) (proto.Message, bool, error) {
	return c.read(id, c.prefix, c.deserializeFunc)
}

func (c *crudImpl) ReadPartial(id string) (proto.Message, bool, error) {
	return c.read(id, c.partialPrefix, c.partialDeserializeFunc)
}

func (c *crudImpl) Exists(id string) (exists bool, err error) {
	err = c.db.View(func(tx *badger.Txn) error {
		key := badgerhelper.GetBucketKey(c.prefix, []byte(id))
		_, err := tx.Get(key)
		if err == nil {
			exists = true
			return nil
		}
		if err == badger.ErrKeyNotFound {
			return nil
		}
		return err
	})
	return
}

func (c *crudImpl) readBatch(prefix []byte, deserializer Deserializer, ids []string) (msgs []proto.Message, indices []int, err error) {
	err = c.db.View(func(tx *badger.Txn) error {
		for idx := range ids {
			msg, err := read(tx, prefix, deserializer, ids[idx])
			if err != nil {
				if err == badger.ErrKeyNotFound {
					indices = append(indices, idx)
					continue
				}
				return err
			}
			msgs = append(msgs, msg)
		}
		return nil
	})
	return
}

func (c *crudImpl) ReadBatch(ids []string) (msgs []proto.Message, indices []int, err error) {
	return c.readBatch(c.prefix, c.deserializeFunc, ids)
}

func (c *crudImpl) ReadBatchPartial(ids []string) (msgs []proto.Message, indices []int, err error) {
	return c.readBatch(c.partialPrefix, c.partialDeserializeFunc, ids)
}

func (c *crudImpl) readAll(prefix []byte, deserializer Deserializer) (msgs []proto.Message, err error) {
	foreachOptions := badgerhelper.ForEachOptions{
		IteratorOptions: badgerhelper.DefaultIteratorOptions(),
	}

	err = c.db.View(func(tx *badger.Txn) error {
		err := badgerhelper.BucketForEach(tx, prefix, foreachOptions, func(k, v []byte) error {
			msg, err := deserializer(v)
			if err != nil {
				return err
			}
			msgs = append(msgs, msg)
			return nil
		})
		return err
	})
	return
}

func (c *crudImpl) ReadAll() (msgs []proto.Message, err error) {
	return c.readAll(c.prefix, c.deserializeFunc)
}

func (c *crudImpl) ReadAllPartial() (msgs []proto.Message, err error) {
	return c.readAll(c.partialPrefix, c.partialDeserializeFunc)
}

func (c *crudImpl) update(tx *badger.Txn, id, data []byte, mustExist bool) error {
	if mustExist {
		if _, err := tx.Get(id); err != nil {
			return err
		}
	}
	return tx.Set(id, data)
}

func (c *crudImpl) resolveManyProtoKV(kvs []proto.Message, resolver func(proto.Message) ([]byte, []byte, error)) ([][]byte, [][]byte, error) {
	ids := make([][]byte, 0, len(kvs))
	data := make([][]byte, 0, len(kvs))

	for _, kv := range kvs {
		id, d, err := resolver(kv)
		if err != nil {
			return nil, nil, err
		}
		ids = append(ids, id)
		data = append(data, d)
	}
	return ids, data, nil
}

func (c *crudImpl) resolveMsgBytes(msg proto.Message) ([]byte, []byte, error) {
	bytes, err := proto.Marshal(msg)
	if err != nil {
		return nil, nil, err
	}
	key := badgerhelper.GetBucketKey(c.prefix, c.keyFunc(msg))
	return key, bytes, nil
}

func (c *crudImpl) resolvePartialMsgBytes(msg proto.Message) ([]byte, []byte, error) {
	if !c.hasPartial {
		return nil, nil, nil
	}
	partialMsg := c.partialConverter(msg)
	bytes, err := proto.Marshal(partialMsg)
	if err != nil {
		return nil, nil, err
	}
	return c.getPartialKey(string(c.keyFunc(msg))), bytes, nil
}

func (c *crudImpl) runUpdate(msg proto.Message, mustExist bool) error {
	id, data, err := c.resolveMsgBytes(msg)
	if err != nil {
		return err
	}
	partialID, partialData, err := c.resolvePartialMsgBytes(msg)
	if err != nil {
		return err
	}
	err = c.db.Update(func(tx *badger.Txn) error {
		if err := c.update(tx, id, data, mustExist); err != nil {
			return err
		}
		if c.hasPartial {
			// No need to check for exist because we already did so if we needed
			return c.update(tx, partialID, partialData, false)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "error updating %s in badger", string(c.prefix))
	}
	return c.IncTxnCount()
}

func (c *crudImpl) Update(msg proto.Message) error {
	return c.runUpdate(msg, true)
}

func (c *crudImpl) Upsert(msg proto.Message) error {
	return c.runUpdate(msg, false)
}

func (c *crudImpl) updateBatch(msgs []proto.Message, mustExist bool) error {
	ids, data, err := c.resolveManyProtoKV(msgs, c.resolveMsgBytes)
	if err != nil {
		return err
	}
	partialIDs, partialData, err := c.resolveManyProtoKV(msgs, c.resolvePartialMsgBytes)
	if err != nil {
		return err
	}

	for i := 0; i < len(ids); i++ {
		err := c.db.Update(func(tx *badger.Txn) error {
			if err := c.update(tx, ids[i], data[i], mustExist); err != nil {
				return err
			}
			if c.hasPartial {
				if err := c.update(tx, partialIDs[i], partialData[i], false); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return errors.Wrapf(err, "error updating many %s in Badger", string(c.prefix))
		}
	}
	return c.txnHelper.IncTxnCount()
}

func (c *crudImpl) UpdateBatch(msgs []proto.Message) error {
	return c.updateBatch(msgs, true)
}

func (c *crudImpl) UpsertBatch(msgs []proto.Message) error {
	return c.updateBatch(msgs, false)
}

func (c *crudImpl) Delete(id string) error {
	key := c.getKey(id)
	partialKey := c.getPartialKey(id)
	err := c.db.Update(func(tx *badger.Txn) error {
		if err := tx.Delete(key); err != nil {
			return err
		}
		if c.hasPartial {
			return tx.Delete(partialKey)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "error deleting %s in badger", c.prefixString)
	}
	return c.IncTxnCount()
}

func (c *crudImpl) DeleteBatch(ids []string) error {
	keys := make([][]byte, 0, len(ids))
	partialKeys := make([][]byte, 0, len(ids))
	for _, i := range ids {
		keys = append(keys, badgerhelper.GetBucketKey(c.prefix, []byte(i)))
		partialKeys = append(partialKeys, badgerhelper.GetBucketKey(c.partialPrefix, []byte(i)))
	}

	batch := c.db.NewWriteBatch()
	defer batch.Cancel()

	for i := 0; i < len(keys); i++ {
		if err := batch.Delete(keys[i]); err != nil {
			return errors.Wrapf(err, "error deleting keys in %s", c.prefixString)
		}
		if c.hasPartial {
			if err := batch.Delete(partialKeys[i]); err != nil {
				return errors.Wrapf(err, "error deleting partial keys in %s", c.prefixString)
			}
		}
	}
	if err := batch.Flush(); err != nil {
		return errors.Wrapf(err, "error flushing batch in %s", c.prefixString)
	}

	return c.IncTxnCount()
}

func (c *crudImpl) GetTxnCount() uint64 {
	return c.txnHelper.GetTxnCount()
}

func (c *crudImpl) IncTxnCount() error {
	return errors.Wrapf(c.txnHelper.IncTxnCount(), "error incrementing txn count in %s", string(c.prefixString))
}

func (c *crudImpl) GetKeys() ([]string, error) {
	var keys []string
	err := c.db.View(func(tx *badger.Txn) error {
		return badgerhelper.BucketKeyForEach(tx, c.prefix, badgerhelper.ForEachOptions{StripKeyPrefix: true}, func(k []byte) error {
			keys = append(keys, string(k))
			return nil
		})
	})
	return keys, err
}
