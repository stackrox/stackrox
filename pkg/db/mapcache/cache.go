package mapcache

import (
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/db"
	"github.com/stackrox/rox/pkg/sync"
)

// NewMapCache takes a db crud and key func and generates a fully in memory cache that wraps the crud interface
// NOTE: This cache expects AT MOST one writer per key. This assumption allows us to avoid taking a lock around the
// cache and the database. Instead, we simply need to lock map operations
func NewMapCache(db db.Crud, keyFunc func(msg proto.Message) []byte) (db.Crud, error) {
	impl := &cacheImpl{
		db:      db,
		keyFunc: keyFunc,

		cache: make(map[string]proto.Message),
	}

	if err := impl.populate(); err != nil {
		return nil, err
	}
	return impl, nil
}

type cacheImpl struct {
	db db.Crud

	keyFunc func(msg proto.Message) []byte
	cache   map[string]proto.Message
	lock    sync.RWMutex
}

func (c *cacheImpl) addNoLock(msg proto.Message) {
	c.cache[string(c.keyFunc(msg))] = proto.Clone(msg)
}

func (c *cacheImpl) populate() error {
	// Locking isn't strictly necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.db.WalkAllWithID(func(id []byte, msg proto.Message) error {
		// No need to clone objects pulled directly from the DB
		c.cache[string(id)] = msg
		return nil
	})
}

func (c *cacheImpl) Count() (int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.cache), nil
}

func (c *cacheImpl) Exists(id string) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.cache[id]
	return ok, nil
}

func (c *cacheImpl) GetKeys() ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]string, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}
	return keys, nil
}

func (c *cacheImpl) Get(id string) (proto.Message, bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	msg, ok := c.cache[id]
	if !ok {
		return nil, false, nil
	}
	return proto.Clone(msg), true, nil
}

func (c *cacheImpl) GetMany(ids []string) ([]proto.Message, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	msgs := make([]proto.Message, 0, len(ids))
	var missingIndices []int
	for i, id := range ids {
		msg, ok := c.cache[id]
		if !ok {
			missingIndices = append(missingIndices, i)
			continue
		}
		msgs = append(msgs, proto.Clone(msg))
	}
	return msgs, missingIndices, nil
}

func (c *cacheImpl) Walk(fn func(msg proto.Message) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, msg := range c.cache {
		if err := fn(proto.Clone(msg)); err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheImpl) WalkAllWithID(fn func(id []byte, msg proto.Message) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for id, msg := range c.cache {
		if err := fn([]byte(id), proto.Clone(msg)); err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheImpl) Upsert(msg proto.Message) error {
	if err := c.db.Upsert(msg); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.addNoLock(msg)
	return nil
}

func (c *cacheImpl) UpsertMany(msgs []proto.Message) error {
	if err := c.db.UpsertMany(msgs); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, msg := range msgs {
		c.addNoLock(msg)
	}
	return nil
}

func (c *cacheImpl) UpsertWithID(id string, msg proto.Message) error {
	if err := c.db.UpsertWithID(id, msg); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache[id] = proto.Clone(msg)
	return nil
}

func (c *cacheImpl) UpsertManyWithIDs(ids []string, msgs []proto.Message) error {
	if len(ids) != len(msgs) {
		return errors.Errorf("length(ids) %d does not match len(msgs) %d", len(ids), len(msgs))
	}

	if err := c.db.UpsertManyWithIDs(ids, msgs); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for i, id := range ids {
		c.cache[id] = proto.Clone(msgs[i])
	}
	return nil
}

func (c *cacheImpl) Delete(id string) error {
	if err := c.db.Delete(id); err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.cache, id)

	return nil
}

func (c *cacheImpl) DeleteMany(ids []string) error {
	if err := c.db.DeleteMany(ids); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, id := range ids {
		delete(c.cache, id)
	}
	return nil
}
