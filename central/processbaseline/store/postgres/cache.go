package postgres

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sync"
)

// NewMapCache takes a db crud and key func and generates a fully in memory cache that wraps the crud interface
// NOTE: This cache expects AT MOST one writer per key. This assumption allows us to avoid taking a lock around the
// cache and the database. Instead, we simply need to lock map operations
func NewMapCache(dbStore Store, keyFunc func(msg storage.ProcessBaseline) []byte) (Store, error) {
	impl := &cacheImpl{
		dbStore: dbStore,
		keyFunc: keyFunc,

		cache: make(map[string]*storage.ProcessBaseline),
	}

	if err := impl.populate(); err != nil {
		return nil, err
	}
	return impl, nil
}

type cacheImpl struct {
	dbStore Store

	keyFunc func(obj storage.ProcessBaseline) []byte
	cache   map[string]*storage.ProcessBaseline
	lock    sync.RWMutex
}

func (c *cacheImpl) populate() error {
	// Locking isn't strictly necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.dbStore.Walk(context.Background(), func(obj *storage.ProcessBaseline) error {
		// No need to clone objects pulled directly from the DB
		c.cache[obj.GetId()] = obj
		return nil
	})
}

func (c *cacheImpl) addNoLock(obj *storage.ProcessBaseline) {
	c.cache[obj.GetId()] = obj.Clone()
}

func (c cacheImpl) Upsert(ctx context.Context, obj *storage.ProcessBaseline) error {
	if err := c.dbStore.Upsert(ctx, obj); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.addNoLock(obj)
	return nil
}

func (c cacheImpl) UpsertMany(ctx context.Context, objs []*storage.ProcessBaseline) error {
	if err := c.dbStore.UpsertMany(ctx, objs); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, obj := range objs {
		c.addNoLock(obj)
	}
	return nil
}

func (c cacheImpl) Delete(ctx context.Context, id string) error {
	if err := c.dbStore.Delete(ctx, id); err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.cache, id)

	return nil
}

func (c cacheImpl) DeleteByQuery(ctx context.Context, q *v1.Query) error {
	//TODO implement me
	panic("implement me")
}

func (c cacheImpl) DeleteMany(ctx context.Context, ids []string) error {
	if err := c.dbStore.DeleteMany(ctx, ids); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, id := range ids {
		delete(c.cache, id)
	}
	return nil
}

func (c cacheImpl) Count(ctx context.Context) (int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.cache), nil
}

func (c cacheImpl) Exists(ctx context.Context, id string) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.cache[id]
	return ok, nil
}

func (c cacheImpl) Get(ctx context.Context, id string) (*storage.ProcessBaseline, bool, error) {
	//TODO implement me
	panic("implement me")
}

func (c cacheImpl) GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.ProcessBaseline, error) {
	//TODO implement me
	panic("implement me")
}

func (c cacheImpl) GetMany(ctx context.Context, identifiers []string) ([]*storage.ProcessBaseline, []int, error) {
	//TODO implement me
	panic("implement me")
}

func (c cacheImpl) GetIDs(_ context.Context) ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ids := make([]string, 0, len(c.cache))
	for id := range c.cache {
		ids = append(ids, id)
	}
	return ids, nil
}

func (c cacheImpl) Walk(ctx context.Context, fn func(obj *storage.ProcessBaseline) error) error {
	//TODO implement me
	panic("implement me")
}

func (c cacheImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return c.dbStore.AckKeysIndexed(ctx, keys...)
}

func (c cacheImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return c.dbStore.GetKeysToIndex(ctx)
}

/*


func (c *cacheImpl) GetKeys(ctx context.Context) ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]string, 0, len(c.cache))
	for key := range c.cache {
		keys = append(keys, key)
	}
	return keys, nil
}

func (c *cacheImpl) Get(ctx context.Context, id string) (*storage.ProcessBaseline, bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	msg, ok := c.cache[id]
	if !ok {
		return nil, false, nil
	}
	return proto.Clone(msg), true, nil
}

func (c *cacheImpl) GetMany(ctx context.Context, ids []string) ([]storage.ProcessBaseline, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	msgs := make([]storage.ProcessBaseline, 0, len(ids))
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

func (c *cacheImpl) Walk(ctx context.Context, fn func(msg storage.ProcessBaseline) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, msg := range c.cache {
		if err := fn(proto.Clone(msg)); err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheImpl) WalkAllWithID(ctx context.Context, fn func(id []byte, msg storage.ProcessBaseline) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for id, msg := range c.cache {
		if err := fn([]byte(id), proto.Clone(msg)); err != nil {
			return err
		}
	}
	return nil
}

func (c *cacheImpl) UpsertMany(ctx context.Context, objs []*storage.ProcessBaseline) error {
}

func (c *cacheImpl) UpsertWithID(ctx context.Context, id string, msg storage.ProcessBaseline) error {
	if err := c.dbStore.UpsertWithID(id, msg); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache[id] = proto.Clone(msg)
	return nil
}

func (c *cacheImpl) UpsertManyWithIDs(ctx context.Context, ids []string, msgs []storage.ProcessBaseline) error {
	if len(ids) != len(msgs) {
		return errors.Errorf("length(ids) %d does not match len(msgs) %d", len(ids), len(msgs))
	}

	if err := c.dbStore.UpsertManyWithIDs(ids, msgs); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	for i, id := range ids {
		c.cache[id] = proto.Clone(msgs[i])
	}
	return nil
}

*/
