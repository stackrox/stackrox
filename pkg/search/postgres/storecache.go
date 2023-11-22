package postgres

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	ops "github.com/stackrox/rox/pkg/metrics"
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
	setCacheOperationDurationTime durationTimeSetter,
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
	store := &cachedStore[T, PT]{
		schema:          schema,
		pkGetter:        pkGetter,
		targetResource:  targetResource,
		cache:           make(map[string]PT),
		underlyingStore: underlyingStore,

		setCacheOperationDurationTime: setCacheOperationDurationTime,
	}
	store.cacheLock.Lock()
	defer store.cacheLock.Unlock()
	// Initial population of the cache. Make sure it is in sync with the DB.
	store.repopulateCacheNoLock()
	return store
}

// NewGenericStoreWithCacheAndPermissionChecker returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithCacheAndPermissionChecker[T any, PT unmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	setCacheOperationDurationTime durationTimeSetter,
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
	store := &cachedStore[T, PT]{
		schema:            schema,
		pkGetter:          pkGetter,
		permissionChecker: checker,
		cache:             make(map[string]PT),
		underlyingStore:   underlyingStore,

		setCacheOperationDurationTime: setCacheOperationDurationTime,
	}
	store.cacheLock.Lock()
	defer store.cacheLock.Unlock()
	// Initial population of the cache. Make sure it is in sync with the DB.
	store.repopulateCacheNoLock()
	return store
}

// cachedStore implements subset of Store interface for resources with single ID.
type cachedStore[T any, PT unmarshaler[T]] struct {
	schema                        *walker.Schema
	pkGetter                      primaryKeyGetter[T, PT]
	setCacheOperationDurationTime durationTimeSetter
	permissionChecker             walker.PermissionChecker
	targetResource                permissions.ResourceMetadata
	underlyingStore               Store[T, PT]
	cache                         map[string]PT
	cacheLock                     sync.RWMutex
}

// Upsert saves the current state of an object in storage.
func (c *cachedStore[T, PT]) Upsert(ctx context.Context, obj PT) error {
	dbErr := c.underlyingStore.Upsert(ctx, obj)
	defer c.setCacheOperationDurationTime(time.Now(), ops.Upsert)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		return dbErr
	}
	c.addToCacheNoLock(obj)
	return nil
}

// UpsertMany saves the state of multiple objects in the storage.
func (c *cachedStore[T, PT]) UpsertMany(ctx context.Context, objs []PT) error {
	dbErr := c.underlyingStore.UpsertMany(ctx, objs)
	defer c.setCacheOperationDurationTime(time.Now(), ops.UpdateMany)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		return dbErr
	}
	for _, obj := range objs {
		c.addToCacheNoLock(obj)
	}
	return nil
}

// Delete removes the object associated to the specified ID from the store.
func (c *cachedStore[T, PT]) Delete(ctx context.Context, id string) error {
	dbErr := c.underlyingStore.Delete(ctx, id)
	defer c.setCacheOperationDurationTime(time.Now(), ops.Remove)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		return dbErr
	}
	delete(c.cache, id)
	return nil
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (c *cachedStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	dbErr := c.underlyingStore.DeleteMany(ctx, identifiers)
	defer c.setCacheOperationDurationTime(time.Now(), ops.RemoveMany)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	if dbErr != nil {
		return dbErr
	}
	for _, id := range identifiers {
		delete(c.cache, id)
	}
	return nil
}

// Exists tells whether the ID exists in the store.
func (c *cachedStore[T, PT]) Exists(ctx context.Context, id string) (bool, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.Exists)
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	obj, found := c.cache[id]
	if !found {
		return false, nil
	}
	return c.isReadAllowed(ctx, obj), nil
}

// Count returns the number of objects in the store.
func (c *cachedStore[T, PT]) Count(ctx context.Context) (int, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.Count)
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

// Get returns the object, if it exists from the store.
func (c *cachedStore[T, PT]) Get(ctx context.Context, id string) (PT, bool, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.Get)
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

// GetMany returns the objects specified by the IDs from the store as well as the index in the missing indices slice.
func (c *cachedStore[T, PT]) GetMany(ctx context.Context, identifiers []string) ([]PT, []int, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetMany)
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

// Walk iterates over all the objects in the store and applies the closure.
func (c *cachedStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	return c.walkCacheNoLock(ctx, fn)
}

// GetByQuery returns the objects from the store matching the query.
func (c *cachedStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	// defer c.setCacheOperationDurationTime(time.Now(), ops.GetByQuery)
	return c.underlyingStore.GetByQuery(ctx, query)
}

// DeleteByQuery removes the objects from the store based on the passed query.
func (c *cachedStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	objs, dbFetchErr := c.underlyingStore.GetByQuery(ctx, query)
	if dbFetchErr != nil {
		return dbFetchErr
	}
	identifiersToRemove := make([]string, 0, len(objs))
	for _, obj := range objs {
		identifiersToRemove = append(identifiersToRemove, c.pkGetter(obj))
	}
	dbRemoveErr := c.underlyingStore.DeleteMany(ctx, identifiersToRemove)
	if dbRemoveErr != nil {
		return dbRemoveErr
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Remove)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, id := range identifiersToRemove {
		delete(c.cache, id)
	}
	return nil
}

// GetIDs returns all the IDs for the store.
func (c *cachedStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetAll)
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

// GetAll retrieves all objects from the store.
//
// Deprecated: This can be dangerous on high cardinality stores consider Walk instead.
func (c *cachedStore[T, PT]) GetAll(ctx context.Context) ([]PT, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetAll)
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

func (c *cachedStore[T, PT]) walkCacheNoLock(ctx context.Context, fn func(obj PT) error) error {
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

func (c *cachedStore[T, PT]) isReadAllowed(ctx context.Context, obj PT) bool {
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
		switch data := interfaceObj.(type) {
		case *storage.NamespaceMetadata:
			scopeChecker = scopeChecker.ClusterID(data.GetClusterId())
			scopeChecker = scopeChecker.Namespace(data.GetName())
		case sac.NamespaceScopedObject:
			scopeChecker = scopeChecker.ForNamespaceScopedObject(data)
		}
	case permissions.ClusterScope:
		var interfaceObj interface{}
		interfaceObj = obj
		switch data := interfaceObj.(type) {
		case *storage.Cluster:
			scopeChecker = scopeChecker.ClusterID(data.GetId())
		case sac.ClusterScopedObject:
			scopeChecker = scopeChecker.ForClusterScopedObject(data)
		}
	}
	return scopeChecker.IsAllowed()
}

func (c *cachedStore[T, PT]) hasPermissionsChecker() bool {
	return c.permissionChecker != nil
}

func (c *cachedStore[T, PT]) repopulateCacheNoLock() {
	c.cache = make(map[string]PT)
	_ = c.underlyingStore.Walk(sac.WithAllAccess(context.Background()), func(obj PT) error {
		c.cache[c.pkGetter(obj)] = obj
		return nil
	})
}

func (c *cachedStore[T, PT]) addToCacheNoLock(obj PT) {
	c.cache[c.pkGetter(obj)] = obj
}
