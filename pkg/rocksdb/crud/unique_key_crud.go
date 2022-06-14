package generic

import (
	"bytes"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/rocksdb"
	"github.com/stackrox/rox/pkg/sync"
)

type uniqueKeyCRUD struct {
	db.Crud

	lock          sync.Mutex
	keyFunc       KeyFunc
	uniqueKeyFunc KeyFunc
}

// NewUniqueKeyCRUD returns a new Crud instance for the given bucket reference, but ensures that no other object in the bucket has the same
// unique key as defined by the uniqueKeyFunc
func NewUniqueKeyCRUD(db *rocksdb.RocksDB, prefix []byte, keyFunc KeyFunc, alloc AllocFunc, uniqueKeyFunc KeyFunc, trackIndex bool) db.Crud {
	return &uniqueKeyCRUD{
		Crud:          NewCRUD(db, prefix, keyFunc, alloc, trackIndex),
		keyFunc:       keyFunc,
		uniqueKeyFunc: uniqueKeyFunc,
	}
}

func (c *uniqueKeyCRUD) checkForConflicts(newID []byte, newMsg proto.Message) error {
	newUniqueKey := c.uniqueKeyFunc(newMsg)
	err := c.Crud.WalkAllWithID(func(id []byte, msg proto.Message) error {
		if bytes.Equal(newID, id) {
			return nil
		}
		if bytes.Equal(newUniqueKey, c.uniqueKeyFunc(msg)) {
			return errors.Wrapf(errox.AlreadyExists, "unique key conflict between %s and existing %s on value: %s", newID, id, newUniqueKey)
		}
		return nil
	})
	return err
}

func (c *uniqueKeyCRUD) Upsert(msg proto.Message) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.checkForConflicts(c.keyFunc(msg), msg); err != nil {
		return err
	}

	return c.Crud.Upsert(msg)
}

func (c *uniqueKeyCRUD) UpsertMany(msgs []proto.Message) error {
	// Check for conflicts amongst the messages being inserted
	conflictMap := make(map[string]string)
	for _, msg := range msgs {
		key := c.keyFunc(msg)
		uniqueKey := string(c.uniqueKeyFunc(msg))
		if existingID, ok := conflictMap[uniqueKey]; ok {
			return errors.Errorf("conflict in batch upsert between %s and %s with unique key conflict: %s", key, existingID, uniqueKey)
		}
		conflictMap[uniqueKey] = string(key)
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for conflicts with existing proto messages in DB
	for _, msg := range msgs {
		if err := c.checkForConflicts(c.keyFunc(msg), msg); err != nil {
			return err
		}
	}

	return c.Crud.UpsertMany(msgs)
}

func (c *uniqueKeyCRUD) UpsertWithID(id string, msg proto.Message) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.checkForConflicts([]byte(id), msg); err != nil {
		return err
	}

	return c.Crud.UpsertWithID(id, msg)
}

func (c *uniqueKeyCRUD) UpsertManyWithIDs(ids []string, msgs []proto.Message) error {
	if len(ids) != len(msgs) {
		return errors.Errorf("len(ids) %d does not match len(msgs) %d", len(ids), len(msgs))
	}

	// Check for conflicts amongst the messages being inserted
	conflictMap := make(map[string]string)
	for idx, msg := range msgs {
		uniqueKey := string(c.uniqueKeyFunc(msg))
		if existingID, ok := conflictMap[uniqueKey]; ok {
			return errors.Errorf("conflict in batch upsert between %s and %s with unique key conflict: %s", ids[idx], existingID, uniqueKey)
		}
		conflictMap[uniqueKey] = ids[idx]
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for conflicts with existing proto messages in DB
	for idx, msg := range msgs {
		if err := c.checkForConflicts([]byte(ids[idx]), msg); err != nil {
			return err
		}
	}

	return c.Crud.UpsertManyWithIDs(ids, msgs)
}
