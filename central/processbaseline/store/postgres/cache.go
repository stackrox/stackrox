package postgres

import (
	"context"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

// NewWithCache takes a postgres db and generates a store with in memory cache
// Note: This cache bypasses the SAC check and relies on the SAC check on datastore.
// If it is not true, the cache may not functional correctly.
func NewWithCache(dbStore Store) (Store, error) {
	impl := &cacheImpl{
		dbStore: dbStore,
		cache:   make(map[string]*storage.ProcessBaseline),
	}

	if err := impl.populate(); err != nil {
		return nil, err
	}
	return impl, nil
}

type cacheImpl struct {
	dbStore Store

	cache map[string]*storage.ProcessBaseline
	lock  sync.RWMutex
}

func (c *cacheImpl) populate() error {
	// Locking isn't strictly necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.dbStore.Walk(sac.WithAllAccess(context.Background()), func(obj *storage.ProcessBaseline) error {
		// No need to clone objects pulled directly from the DB
		c.cache[obj.GetId()] = obj
		return nil
	})
}

func (c *cacheImpl) addNoLock(obj *storage.ProcessBaseline) {
	c.cache[obj.GetId()] = obj.Clone()
}

func (c *cacheImpl) Upsert(ctx context.Context, obj *storage.ProcessBaseline) error {
	if err := c.dbStore.Upsert(ctx, obj); err != nil {
		return err
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	c.addNoLock(obj)
	return nil
}

func (c *cacheImpl) UpsertMany(ctx context.Context, objs []*storage.ProcessBaseline) error {
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

func (c *cacheImpl) Delete(ctx context.Context, id string) error {
	if err := c.dbStore.Delete(ctx, id); err != nil {
		return err
	}

	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.cache, id)

	return nil
}

func (c *cacheImpl) DeleteByQuery(_ context.Context, _ *v1.Query) ([]string, error) {
	// TODO implement me
	panic("implement me")
}

func (c *cacheImpl) DeleteMany(ctx context.Context, ids []string) error {
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

func (c *cacheImpl) Count(_ context.Context) (int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.cache), nil
}

func (c *cacheImpl) Exists(_ context.Context, id string) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.cache[id]
	return ok, nil
}

func (c *cacheImpl) Get(_ context.Context, id string) (*storage.ProcessBaseline, bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	obj, ok := c.cache[id]
	if !ok {
		return nil, false, nil
	}
	return obj.Clone(), true, nil
}

func (c *cacheImpl) GetByQuery(ctx context.Context, query *v1.Query) ([]*storage.ProcessBaseline, error) {
	// Call dbStore function directly. It is not in use as of current time so add a reminder here.
	utils.Should(errors.New("Engineering alert: this function is not optimized."))
	return c.dbStore.GetByQuery(ctx, query)
}

func (c *cacheImpl) GetMany(_ context.Context, ids []string) ([]*storage.ProcessBaseline, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	objs := make([]*storage.ProcessBaseline, 0, len(ids))
	var missingIndices []int
	for i, id := range ids {
		obj, ok := c.cache[id]
		if !ok {
			missingIndices = append(missingIndices, i)
			continue
		}
		objs = append(objs, obj.Clone())
	}
	return objs, missingIndices, nil
}

func (c *cacheImpl) GetIDs(_ context.Context) ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	ids := make([]string, 0, len(c.cache))
	for id := range c.cache {
		ids = append(ids, id)
	}
	return ids, nil
}

func (c *cacheImpl) Walk(_ context.Context, fn func(obj *storage.ProcessBaseline) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	for _, obj := range c.cache {
		if err := fn(obj.Clone()); err != nil {
			return err
		}
	}
	return nil
}
