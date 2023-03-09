package postgres

import (
	"context"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// maxCacheSize = 1024 * 1024 // Target value
	maxCacheSize = 20 // low value testing correctness
)

// NewWithCache takes a postgres db and generates a store with in memory cache
// Note: This cache bypasses the SAC check and relies on the SAC check on datastore.
// If it is not true, the cache may not functional correctly.
func NewWithCache(dbStore Store) (Store, error) {
	impl := &cacheImpl{
		dbStore: dbStore,
	}
	var err error

	impl.cache, err = lru.NewWithEvict(maxCacheSize, func(key string, value *storage.ProcessBaseline) {
		impl.overflowIDs.Add(key)
	})

	if err != nil {
		return nil, err
	}

	if err = impl.populate(); err != nil {
		return nil, err
	}
	return impl, nil
}

type cacheImpl struct {
	dbStore Store

	cache       *lru.Cache[string, *storage.ProcessBaseline]
	overflowIDs set.StringSet
	lock        sync.RWMutex
}

func (c *cacheImpl) populate() error {
	// Locking isn't strictly necessary
	c.lock.Lock()
	defer c.lock.Unlock()

	return c.dbStore.Walk(sac.WithAllAccess(context.Background()), func(obj *storage.ProcessBaseline) error {
		c.cache.Add(obj.GetId(), obj)
		return nil
	})
}

func (c *cacheImpl) addNoLock(obj *storage.ProcessBaseline) {
	c.cache.Add(obj.GetId(), obj.Clone())
	c.overflowIDs.Remove(obj.GetId())
}

func (c *cacheImpl) deleteNoLock(id string) {
	c.cache.Remove(id)
	c.overflowIDs.Remove(id)
}

func (c *cacheImpl) Upsert(ctx context.Context, obj *storage.ProcessBaseline) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.dbStore.Upsert(ctx, obj); err != nil {
		return err
	}

	c.addNoLock(obj)
	c.sanityCheckNoLock(ctx)
	return nil
}

func (c *cacheImpl) UpsertMany(ctx context.Context, objs []*storage.ProcessBaseline) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if err := c.dbStore.UpsertMany(ctx, objs); err != nil {
		return err
	}

	for _, obj := range objs {
		c.addNoLock(obj)
	}
	c.sanityCheckNoLock(ctx)
	return nil
}

func (c *cacheImpl) Delete(ctx context.Context, id string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.dbStore.Delete(ctx, id); err != nil {
		return err
	}

	c.deleteNoLock(id)

	c.sanityCheckNoLock(ctx)
	return nil
}

func (c *cacheImpl) DeleteByQuery(ctx context.Context, q *v1.Query) error {
	// TODO implement me
	panic("implement me")
}

func (c *cacheImpl) DeleteMany(ctx context.Context, ids []string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.dbStore.DeleteMany(ctx, ids); err != nil {
		return err
	}

	for _, id := range ids {
		c.deleteNoLock(id)
	}
	c.sanityCheckNoLock(ctx)
	return nil
}

func (c *cacheImpl) Count(ctx context.Context) (int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	c.sanityCheckNoLock(ctx)
	return c.cache.Len() + c.overflowIDs.Cardinality(), nil
}

func (c *cacheImpl) Exists(ctx context.Context, id string) (bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	c.sanityCheckNoLock(ctx)
	return c.cache.Contains(id) || c.overflowIDs.Contains(id), nil
}

func (c *cacheImpl) Get(ctx context.Context, id string) (*storage.ProcessBaseline, bool, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	c.sanityCheckNoLock(ctx)
	if c.overflowIDs.Contains(id) {
		if c.cache.Contains(id) {
			utils.Should(errors.Errorf("cache and overflow contains both %s", id))
		}
		obj, exists, err := c.dbStore.Get(ctx, id)
		if err == nil {
			return nil, false, err
		}
		if !exists {
			utils.Should(errors.Errorf("cache inconsistency for missing entry in database with id %s", id))
		}
		c.cache.Add(id, obj)
		c.sanityCheckNoLock(ctx)
		return obj.Clone(), exists, err
	}

	obj, ok := c.cache.Get(id)
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

func (c *cacheImpl) GetMany(ctx context.Context, ids []string) ([]*storage.ProcessBaseline, []int, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	c.sanityCheckNoLock(ctx)
	objs := make([]*storage.ProcessBaseline, 0, len(ids))
	var missingIndices []int
	var idsToFetch []string
	for i, id := range ids {
		if c.overflowIDs.Contains(id) {
			idsToFetch = append(idsToFetch, id)
			continue
		}
		obj, ok := c.cache.Get(id)
		if !ok {
			missingIndices = append(missingIndices, i)
			continue
		}
		objs = append(objs, obj.Clone())
	}

	fetched, missing, err := c.dbStore.GetMany(ctx, idsToFetch)
	if err != nil {
		return nil, nil, err
	}
	if len(missing) != 0 {
		utils.Should(errors.Errorf("unexpected missing elements in fetch %v, missing %v", idsToFetch, missing))
	}

	for _, fetchedObj := range fetched {
		c.addNoLock(fetchedObj)
		objs = append(objs, fetchedObj)
	}
	c.sanityCheckNoLock(ctx)

	return objs, missingIndices, nil
}

func (c *cacheImpl) GetIDs(ctx context.Context) ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	c.sanityCheckNoLock(ctx)
	ids := c.cache.Keys()
	ids = append(ids, c.overflowIDs.AsSlice()...)

	c.sanityCheckNoLock(ctx)
	return ids, nil
}

func (c *cacheImpl) Walk(ctx context.Context, fn func(obj *storage.ProcessBaseline) error) error {
	c.lock.RLock()
	defer c.lock.RUnlock()
	c.sanityCheckNoLock(ctx)

	for _, id := range c.cache.Keys() {
		// Since Walk access every entry, no bother to update access of cache
		obj, _ := c.cache.Peek(id)
		if err := fn(obj); err != nil {
			return err
		}
	}

	if c.overflowIDs.Cardinality() > 0 {
		objs, missing, err := c.dbStore.GetMany(ctx, c.overflowIDs.AsSlice())
		if err != nil {
			return nil
		}
		if len(missing) != 0 {
			utils.Should(errors.Errorf("unexpected missing entries: %+v", missing))
		}
		for _, obj := range objs {
			if err = fn(obj); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *cacheImpl) AckKeysIndexed(ctx context.Context, keys ...string) error {
	return c.dbStore.AckKeysIndexed(ctx, keys...)
}

func (c *cacheImpl) GetKeysToIndex(ctx context.Context) ([]string, error) {
	return c.dbStore.GetKeysToIndex(ctx)
}

func (c *cacheImpl) sanityCheckNoLock(ctx context.Context) {
	count, err := c.dbStore.Count(sac.WithAllAccess(context.Background()))
	utils.Should(err)
	if overlaps := c.overflowIDs.Intersect(set.NewStringSet(c.cache.Keys()...)); overlaps.Cardinality() != 0 {
		utils.Should(errors.Errorf("unexpected overlap %v", overlaps))
	}
	if count != c.cache.Len()+c.overflowIDs.Cardinality() {
		utils.Should(errors.Errorf("inconsistent cache count db %d cache %d", count, c.cache.Len()+c.overflowIDs.Cardinality()))
	}
}
