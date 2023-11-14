package postgres

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sync"
)

// NewGenericStoreWithCache returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithCache[T any, PT unmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	upsertAllowed upsertChecker[T, PT],
	targetResource permissions.ResourceMetadata,
) Store[T, PT] {
	underlyingStore := NewGenericStore[T, PT](
		db,
		schema,
		pkGetter,
		insertInto,
		copyFromObj,
		setAcquireDBConnDuration,
		setPostgresOperationDurationTime,
		upsertAllowed,
		targetResource,
	)
	store := &CachedStore[T, PT]{
		schema:          schema,
		pkGetter:        pkGetter,
		targetResource:  targetResource,
		cache:           make(map[string]PT),
		useCache:        true,
		underlyingStore: underlyingStore,
	}
	store.cacheLock.Lock()
	defer store.cacheLock.Unlock()
	store.resetCacheNoLock()
	return store
}

// NewGenericStoreWithPermissionCheckerWithCache returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithPermissionCheckerWithCache[T any, PT unmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	checker walker.PermissionChecker,
) Store[T, PT] {
	underlyingStore := NewGenericStoreWithPermissionChecker[T, PT](
		db,
		schema,
		pkGetter,
		insertInto,
		copyFromObj,
		setAcquireDBConnDuration,
		setPostgresOperationDurationTime,
		checker,
	)
	store := &CachedStore[T, PT]{
		schema:            schema,
		pkGetter:          pkGetter,
		permissionChecker: checker,
		cache:             make(map[string]PT),
		useCache:          true,
		underlyingStore:   underlyingStore,
	}
	store.cacheLock.Lock()
	defer store.cacheLock.Unlock()
	store.resetCacheNoLock()
	return store
}

// CachedStore implements subset of Store interface for resources with single ID.
type CachedStore[T any, PT unmarshaler[T]] struct {
	mutex                         sync.RWMutex
	schema                        *walker.Schema
	pkGetter                      primaryKeyGetter[T, PT]
	setCacheOperationDurationTime durationTimeSetter
	permissionChecker             walker.PermissionChecker
	targetResource                permissions.ResourceMetadata
	underlyingStore               Store[T, PT]
	cache                         map[string]PT
	useCache                      bool
	cacheLock                     sync.RWMutex
}

func (c *CachedStore[T, PT]) Upsert(ctx context.Context, obj PT) error {
	dbErr := c.underlyingStore.Upsert(ctx, obj)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		c.resetCacheNoLock()
		return dbErr
	}
	c.addToCacheNoLock(obj)
	return nil
}

func (c *CachedStore[T, PT]) UpsertMany(ctx context.Context, objs []PT) error {
	dbErr := c.underlyingStore.UpsertMany(ctx, objs)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		c.resetCacheNoLock()
		return dbErr
	}
	for _, obj := range objs {
		c.addToCacheNoLock(obj)
	}
	return nil
}

func (c *CachedStore[T, PT]) Delete(ctx context.Context, id string) error {
	dbErr := c.underlyingStore.Delete(ctx, id)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		c.resetCacheNoLock()
		return dbErr
	}
	delete(c.cache, id)
	return nil
}

func (c *CachedStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	dbErr := c.underlyingStore.DeleteMany(ctx, identifiers)
	c.cacheLock.Lock()
	defer c.cacheLock.RUnlock()
	if dbErr != nil {
		c.resetCacheNoLock()
		return dbErr
	}
	for _, id := range identifiers {
		delete(c.cache, id)
	}
	return nil
}

func (c *CachedStore[T, PT]) Exists(ctx context.Context, id string) (bool, error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	obj, found := c.cache[id]
	if !found {
		return false, nil
	}
	return c.isReadAllowed(ctx, obj), nil
}

func (c *CachedStore[T, PT]) Count(ctx context.Context) (int, error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	count := 0
	walkErr := c.walkCacheNoLock(ctx, func(obj PT) error {
		count++
		return nil
	})
	if walkErr != nil {
		return 0, walkErr
	}
	return count, nil
}

func (c *CachedStore[T, PT]) Get(ctx context.Context, id string) (PT, bool, error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	obj, found := c.cache[id]
	if !found {
		return nil, false, nil
	}
	if !c.isReadAllowed(ctx, obj) {
		return nil, false, nil
	}
	return obj, true, nil
}

func (c *CachedStore[T, PT]) GetMany(ctx context.Context, identifiers []string) ([]PT, []int, error) {
	if len(identifiers) == 0 {
		return nil, nil, nil
	}
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	results := make([]PT, 0, len(identifiers))
	misses := make([]int, 0, len(identifiers))
	for ix, id := range identifiers {
		obj, found := c.cache[id]
		if !found {
			misses = append(misses, ix)
			continue
		}
		if !c.isReadAllowed(ctx, obj) {
			misses = append(misses, ix)
			continue
		}
		results = append(results, obj)
	}
	return results, misses, nil
}

func (c *CachedStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	return c.walkCacheNoLock(ctx, fn)
}

func (c *CachedStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	return c.underlyingStore.GetByQuery(ctx, query)
}

func (c *CachedStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	dbErr := c.underlyingStore.DeleteByQuery(ctx, query)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.resetCacheNoLock()
	return dbErr
}

func (c *CachedStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	result := make([]string, 0, len(c.cache))
	walkErr := c.walkCacheNoLock(ctx, func(obj PT) error {
		result = append(result, c.pkGetter(obj))
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return result, nil
}

func (c *CachedStore[T, PT]) GetAll(ctx context.Context) ([]PT, error) {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	result := make([]PT, 0, len(c.cache))
	walkErr := c.walkCacheNoLock(ctx, func(obj PT) error {
		result = append(result, obj)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return result, nil
}

func (c *CachedStore[T, PT]) walkCacheNoLock(ctx context.Context, fn func(obj PT) error) error {
	for _, obj := range c.cache {
		if !c.isReadAllowed(ctx, obj) {
			continue
		}
		fnErr := fn(obj)
		if fnErr != nil {
			return fnErr
		}
	}
	return nil
}

func (c *CachedStore[T, PT]) isReadAllowed(ctx context.Context, obj PT) bool {
	if c.hasPermissionsChecker() {
		allowed, err := c.permissionChecker.ReadAllowed(ctx)
		if err != nil {
			return false
		}
		return allowed
	}
	scopeChecker := sac.GlobalAccessScopeChecker(ctx)
	scopeChecker = scopeChecker.AccessMode(storage.Access_READ_ACCESS)
	scopeChecker = scopeChecker.Resource(c.targetResource)
	switch c.targetResource.GetScope() {
	case permissions.NamespaceScope:
		var interfaceObj interface{}
		interfaceObj = obj
		namespaceScopedObj := interfaceObj.(sac.NamespaceScopedObject)
		scopeChecker = scopeChecker.ForNamespaceScopedObject(namespaceScopedObj)
	case permissions.ClusterScope:
		var interfaceObj interface{}
		interfaceObj = obj
		clusterScopedObj := interfaceObj.(sac.ClusterScopedObject)
		scopeChecker = scopeChecker.ForClusterScopedObject(clusterScopedObj)
	}
	return scopeChecker.IsAllowed()
}

func (c *CachedStore[T, PT]) hasPermissionsChecker() bool {
	return c.permissionChecker != nil
}

func (c *CachedStore[T, PT]) resetCacheNoLock() {
	c.cache = make(map[string]PT)
	_ = c.underlyingStore.Walk(sac.WithAllAccess(context.Background()), func(obj PT) error {
		c.cache[c.pkGetter(obj)] = obj
		return nil
	})
}

func (c *CachedStore[T, PT]) addToCacheNoLock(obj PT) {
	c.cache[c.pkGetter(obj)] = obj
}
