package postgres

import (
	"context"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/sync"
)

// NewGenericStoreWithCache returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithCache[T any, PT ClonedUnmarshaler[T]](
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
	// Initial population of the cache. Make sure it is in sync with the DB.
	err := store.populateCache()
	if err != nil {
		// Failed to populate the cache, return the store connected to the DB
		// in order to avoid serving data from a cache not consistent with
		// the underlying database.
		log.Errorf("Failed to populate store cache, using direct store access instead: %v", err)
		return underlyingStore
	}
	return store
}

// NewGenericStoreWithCacheAndPermissionChecker returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGenericStoreWithCacheAndPermissionChecker[T any, PT ClonedUnmarshaler[T]](
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
	// Initial population of the cache. Make sure it is in sync with the DB.
	err := store.populateCache()
	if err != nil {
		// Failed to populate the cache, return the store connected to the DB
		// in order to avoid serving data from a cache not consistent with
		// the underlying database.
		log.Errorf("Failed to populate store cache, using direct store access instead: %v", err)
		return underlyingStore
	}
	return store
}

// cachedStore implements subset of Store interface for resources with single ID.
type cachedStore[T any, PT ClonedUnmarshaler[T]] struct {
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
	err := c.underlyingStore.Upsert(ctx, obj)
	if err != nil {
		return err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Upsert)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.addToCacheNoLock(obj)
	return nil
}

// UpsertMany saves the state of multiple objects in the storage.
func (c *cachedStore[T, PT]) UpsertMany(ctx context.Context, objs []PT) error {
	err := c.underlyingStore.UpsertMany(ctx, objs)
	if err != nil {
		return err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.UpdateMany)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, obj := range objs {
		c.addToCacheNoLock(obj)
	}
	return nil
}

// Delete removes the object associated to the specified ID from the store.
func (c *cachedStore[T, PT]) Delete(ctx context.Context, id string) error {
	obj, found := concurrency.WithRLock2[PT, bool](&c.cacheLock, func() (PT, bool) {
		obj, found := c.cache[id]
		return obj, found
	})
	if !c.isWriteAllowed(ctx, obj) {
		// Special case: the query generator for global scoped resources currently returns sac.ErrResourceAccessDenied
		// when write is not allowed. The cached store needs to respect that behavior for the time being.
		// Ultimately, we have to convey on the behavior: either _always_ return sac.ErrResourceAccessDenied or return
		// nil.
		// TODO(ROX-22408): Once the behavior is fixed, remove this special casing.
		if c.targetResource.GetScope() == permissions.GlobalScope {
			return sac.ErrResourceAccessDenied
		}
		return nil
	}
	if !found {
		return nil
	}
	err := c.underlyingStore.Delete(ctx, id)
	if err != nil {
		return err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Remove)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	delete(c.cache, id)
	return nil
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (c *cachedStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	if len(identifiers) == 0 {
		return nil
	}
	objects := make([]PT, 0, len(identifiers))
	concurrency.WithRLock(&c.cacheLock, func() {
		for _, identifier := range identifiers {
			obj, found := c.cache[identifier]
			if !found {
				continue
			}
			objects = append(objects, obj)
		}
	})
	filteredIDs := make([]string, 0, len(objects))
	for _, obj := range objects {
		if !c.isWriteAllowed(ctx, obj) {
			continue
		}
		filteredIDs = append(filteredIDs, c.pkGetter(obj))
	}
	err := c.underlyingStore.DeleteMany(ctx, filteredIDs)
	if err != nil {
		return err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.RemoveMany)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, id := range filteredIDs {
		delete(c.cache, id)
	}
	return nil
}

// PruneMany removes the objects associated to the specified IDs from the store.
func (c *cachedStore[T, PT]) PruneMany(ctx context.Context, identifiers []string) error {
	if len(identifiers) == 0 {
		return nil
	}

	// Ideally we could use PruneMany, but since a batch of pruning can fail that could lead
	// to inconsistencies with the cache.  So for the cache it is best to continue to using
	// the cachedStore DeleteMany as it does batched deletion at DB level as well as cache synchronization.
	return c.DeleteMany(ctx, identifiers)
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

// Count returns the number of objects in the store matching the query.
func (c *cachedStore[T, PT]) Count(ctx context.Context, q *v1.Query) (int, error) {
	if q == nil || q.EqualVT(search.EmptyQuery()) {
		return c.countFromCache(ctx)
	}
	return c.underlyingStore.Count(ctx, q)
}

func (c *cachedStore[T, PT]) countFromCache(ctx context.Context) (int, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.Count)
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	count := 0
	err := c.walkCacheNoLock(ctx, func(obj PT) error {
		count++
		return nil
	})
	if err != nil {
		return 0, err
	}
	return count, nil
}

// Search searches for objects matching the query.
func (c *cachedStore[T, PT]) Search(ctx context.Context, q *v1.Query) ([]search.Result, error) {
	return c.underlyingStore.Search(ctx, q)
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
	return obj.CloneVT(), true, nil
}

// GetByIDs returns the objects specified by the IDs from the store.
func (c *cachedStore[T, PT]) GetByIDs(ctx context.Context, identifiers []string) ([]PT, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetMany)
	if len(identifiers) == 0 {
		return nil, nil
	}
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	results := make([]PT, 0, len(identifiers))
	for _, id := range identifiers {
		obj, found := c.cache[id]
		if !found || !c.isReadAllowed(ctx, obj) {
			continue
		}
		results = append(results, obj.CloneVT())
	}
	return results, nil
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
	misses := make([]int, 0)
	for idx, id := range identifiers {
		obj, found := c.cache[id]
		if !found {
			misses = append(misses, idx)
			continue
		}
		if !c.isReadAllowed(ctx, obj) {
			misses = append(misses, idx)
			continue
		}
		results = append(results, obj.CloneVT())
	}
	return results, misses, nil
}

// WalkByQuery iterates over all the objects scoped by the query applies the closure.
func (c *cachedStore[T, PT]) WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj PT) error) error {
	if query == nil || query.EqualVT(search.EmptyQuery()) {
		c.cacheLock.RLock()
		defer c.cacheLock.RUnlock()
		return c.walkCacheNoLock(ctx, fn)
	}

	identifiers, err := c.underlyingStore.GetIDsByQuery(ctx, query)
	// Fallback to the underlying store on error.
	if err != nil {
		log.Errorf("Failed to get identifiers by query, falling back to walk results by query: %v", err)
		return c.underlyingStore.WalkByQuery(ctx, query, fn)
	}
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	for _, id := range identifiers {
		if err := ctx.Err(); err != nil {
			return err
		}
		obj, found := c.cache[id]
		if !found {
			log.Warnf("Object %q not found in store cache", id)
			continue
		}
		if !c.isReadAllowed(ctx, obj) {
			continue
		}
		if err := fn(obj); err != nil {
			return err
		}
	}
	return nil
}

// Walk iterates over all the objects in the store and applies the closure.
func (c *cachedStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	return c.walkCacheNoLock(ctx, fn)
}

// GetByQuery returns the objects from the store matching the query.
func (c *cachedStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetByQuery)
	identifiers, err := c.underlyingStore.GetIDsByQuery(ctx, query)
	// Fallback to the underlying store on error.
	if err != nil {
		log.Errorf("Failed to get identifiers by query, falling back to get results by query: %v", err)
		return c.underlyingStore.GetByQuery(ctx, query)
	}
	results := make([]*T, 0, len(identifiers))
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	for _, id := range identifiers {
		obj, found := c.cache[id]
		if !found {
			log.Warnf("Object %q not found in store cache", id)
			continue
		}
		if !c.isReadAllowed(ctx, obj) {
			continue
		}
		results = append(results, obj.CloneVT())
	}
	return results, nil
}

// DeleteByQuery removes the objects from the store based on the passed query.
func (c *cachedStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) ([]string, error) {
	identifiersToRemove, err := c.underlyingStore.DeleteByQuery(ctx, query)
	if err != nil {
		return nil, err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Remove)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, id := range identifiersToRemove {
		delete(c.cache, id)
	}
	return identifiersToRemove, nil
}

// GetIDs returns all the IDs for the store.
func (c *cachedStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetAll)
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	result := make([]string, 0, len(c.cache))
	err := c.walkCacheNoLock(ctx, func(obj PT) error {
		result = append(result, c.pkGetter(obj))
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetIDsByQuery returns the IDs for the store matching the query.
func (c *cachedStore[T, PT]) GetIDsByQuery(ctx context.Context, query *v1.Query) ([]string, error) {
	return c.underlyingStore.GetIDsByQuery(ctx, query)
}

func (c *cachedStore[T, PT]) walkCacheNoLock(ctx context.Context, fn func(obj PT) error) error {
	for _, obj := range c.cache {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !c.isReadAllowed(ctx, obj) {
			continue
		}
		err := fn(obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *cachedStore[T, PT]) isReadAllowed(ctx context.Context, obj PT) bool {
	return c.isActionAllowed(ctx, storage.Access_READ_ACCESS, obj)
}

func (c *cachedStore[T, PT]) isWriteAllowed(ctx context.Context, obj PT) bool {
	return c.isActionAllowed(ctx, storage.Access_READ_WRITE_ACCESS, obj)
}

func (c *cachedStore[T, PT]) isActionAllowed(ctx context.Context, action storage.Access, obj PT) bool {
	if c.hasPermissionsChecker() {
		var allowed bool
		var err error
		switch action {
		case storage.Access_READ_ACCESS:
			allowed, err = c.permissionChecker.ReadAllowed(ctx)
		case storage.Access_READ_WRITE_ACCESS:
			allowed, err = c.permissionChecker.WriteAllowed(ctx)
		default:
			return false
		}
		if err != nil {
			return false
		}
		return allowed
	}
	scopeChecker := sac.GlobalAccessScopeChecker(ctx).AccessMode(action).Resource(c.targetResource)
	var interfaceObj interface{} = obj
	switch c.targetResource.GetScope() {
	case permissions.NamespaceScope:
		switch data := interfaceObj.(type) {
		case *storage.NamespaceMetadata:
			scopeChecker = scopeChecker.ClusterID(data.GetClusterId()).Namespace(data.GetName())
		case *storage.ProcessBaseline:
			scopeChecker = scopeChecker.ClusterID(data.GetKey().GetClusterId()).Namespace(data.GetKey().GetNamespace())
		case sac.NamespaceScopedObject:
			scopeChecker = scopeChecker.ForNamespaceScopedObject(data)
		}
	case permissions.ClusterScope:
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

func (c *cachedStore[T, PT]) populateCache() error {
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.cache = make(map[string]PT)
	return c.underlyingStore.Walk(sac.WithAllAccess(context.Background()), func(obj PT) error {
		c.addToCacheNoLock(obj)
		return nil
	})
}

func (c *cachedStore[T, PT]) addToCacheNoLock(obj PT) {
	c.cache[c.pkGetter(obj)] = obj.CloneVT()
}
