package postgres

import (
	"context"
	"maps"
	"slices"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/postgres"
	"github.com/stackrox/rox/pkg/postgres/walker"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/scoped"
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
	defaultSort *v1.QuerySortOption,
	transformOptionsMap search.OptionsMap,
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
		defaultSort,
		transformOptionsMap,
	)
	if storeCacheDisabled(db) {
		return underlyingStore
	}
	store := &cachedStore[T, PT]{
		schema:          schema,
		pkGetter:        pkGetter,
		targetResource:  targetResource,
		cache:           make(map[string]PT),
		underlyingStore: underlyingStore,

		setCacheOperationDurationTime: setCacheOperationDurationTime,
	}
	// Initial population of the cache. Make sure it is in sync with the DB.
	err := store.initializeCache(db)
	if err != nil {
		// Failed to populate the cache, return the store connected to the DB
		// in order to avoid serving data from a cache not consistent with
		// the underlying database.
		log.Errorf("Failed to populate store cache, using direct store access instead: %v", err)
		return underlyingStore
	}
	return store
}

// NewGloballyScopedGenericStoreWithCache returns new subStore implementation for given resource.
// subStore implements subset of Store operations.
func NewGloballyScopedGenericStoreWithCache[T any, PT ClonedUnmarshaler[T]](
	db postgres.DB,
	schema *walker.Schema,
	pkGetter primaryKeyGetter[T, PT],
	insertInto inserter[T, PT],
	copyFromObj copier[T, PT],
	setAcquireDBConnDuration durationTimeSetter,
	setPostgresOperationDurationTime durationTimeSetter,
	setCacheOperationDurationTime durationTimeSetter,
	targetResource permissions.ResourceMetadata,
	defaultSort *v1.QuerySortOption,
	transformOptionsMap search.OptionsMap,
) Store[T, PT] {
	underlyingStore := NewGloballyScopedGenericStore[T, PT](
		db,
		schema,
		pkGetter,
		insertInto,
		copyFromObj,
		setAcquireDBConnDuration,
		setPostgresOperationDurationTime,
		targetResource,
		defaultSort,
		transformOptionsMap,
	)
	if storeCacheDisabled(db) {
		return underlyingStore
	}
	store := &cachedStore[T, PT]{
		schema:          schema,
		pkGetter:        pkGetter,
		targetResource:  targetResource,
		cache:           make(map[string]PT),
		underlyingStore: underlyingStore,

		setCacheOperationDurationTime: setCacheOperationDurationTime,
	}
	// Initial population of the cache. Make sure it is in sync with the DB.
	err := store.initializeCache(db)
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
	targetResource                permissions.ResourceMetadata
	underlyingStore               Store[T, PT]
	cache                         map[string]PT
	cacheLock                     sync.RWMutex
	mutationGate                  sync.RWMutex
	trusted                       bool
	generation                    uint64
	fullVersion                   uint64
	registration                  *cacheRegistration
	observerLock                  sync.Mutex
	observers                     map[*cacheObserver[PT]]struct{}
	observerCount                 atomic.Int32
	removed                       []PT
}

// CacheEnabled reports whether secondary indexes can use this store's cache mode.
func (c *cachedStore[T, PT]) CacheEnabled() bool {
	if c.registration != nil && !c.registration.coordinator.isConnected() {
		return false
	}
	return concurrency.WithRLock1(&c.cacheLock, func() bool { return c.trusted })
}

// Upsert saves the current state of an object in storage.
func (c *cachedStore[T, PT]) Upsert(ctx context.Context, obj PT) error {
	defer c.beginMutation(ctx)()
	err := c.underlyingStore.Upsert(ctx, obj)
	if err != nil {
		c.mutationFailed()
		return err
	}
	if postgres.HasTxInContext(ctx) {
		return nil
	}
	defer c.signalObservers()
	defer c.setCacheOperationDurationTime(time.Now(), ops.Upsert)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.addToCacheNoLock(obj)
	c.setCacheEntriesGauge()
	return nil
}

// UpsertMany saves the state of multiple objects in the storage.
func (c *cachedStore[T, PT]) UpsertMany(ctx context.Context, objs []PT) error {
	defer c.beginMutation(ctx)()
	err := c.underlyingStore.UpsertMany(ctx, objs)
	if err != nil {
		c.mutationFailed()
		return err
	}
	if postgres.HasTxInContext(ctx) {
		return nil
	}
	defer c.signalObservers()
	defer c.setCacheOperationDurationTime(time.Now(), ops.UpdateMany)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, obj := range objs {
		c.addToCacheNoLock(obj)
	}
	c.setCacheEntriesGauge()
	return nil
}

// Delete removes the object associated to the specified ID from the store.
func (c *cachedStore[T, PT]) Delete(ctx context.Context, id string) error {
	if c.registration != nil || !c.useCache(ctx) {
		_, err := c.deleteByQueryWithIDs(ctx, search.NewQueryBuilder().AddDocIDs(id).ProtoQuery())
		return err
	}
	defer c.beginMutation(ctx)()
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
		c.mutationFailed()
		return err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Remove)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	c.removeFromCacheNoLock(id)
	c.setCacheEntriesGauge()
	return nil
}

// DeleteMany removes the objects associated to the specified IDs from the store.
func (c *cachedStore[T, PT]) DeleteMany(ctx context.Context, identifiers []string) error {
	if len(identifiers) == 0 {
		return nil
	}
	if c.registration != nil || !c.useCache(ctx) {
		_, err := c.deleteByQueryWithIDs(ctx, search.NewQueryBuilder().AddDocIDs(identifiers...).ProtoQuery())
		return err
	}
	defer c.beginMutation(ctx)()
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
		c.mutationFailed()
		return err
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.RemoveMany)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, id := range filteredIDs {
		c.removeFromCacheNoLock(id)
	}
	c.setCacheEntriesGauge()
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
	if !c.useCache(ctx) {
		return c.underlyingStore.Exists(ctx, id)
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Exists)
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	obj, found := c.cache[id]
	if !found {
		cacheMissTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "Exists"}).Inc()
		return false, nil
	}
	cacheHitTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "Exists"}).Inc()
	return c.isReadAllowed(ctx, obj), nil
}

// Count returns the number of objects in the store matching the query.
func (c *cachedStore[T, PT]) Count(ctx context.Context, q *v1.Query) (int, error) {
	if c.useCache(ctx) && checkScopeQueries(ctx, q) {
		return c.countFromCache(ctx)
	}
	cacheBypassTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "Count"}).Inc()
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
	if !c.useCache(ctx) {
		return c.underlyingStore.Get(ctx, id)
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.Get)
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	obj, found := c.cache[id]
	if !found {
		cacheMissTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "Get"}).Inc()
		return nil, false, nil
	}
	if !c.isReadAllowed(ctx, obj) {
		return nil, false, nil
	}
	cacheHitTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "Get"}).Inc()
	return obj.CloneVT(), true, nil
}

// GetMany returns the objects specified by the IDs from the store as well as the index in the missing indices slice.
func (c *cachedStore[T, PT]) GetMany(ctx context.Context, identifiers []string) ([]PT, []int, error) {
	if !c.useCache(ctx) {
		return c.underlyingStore.GetMany(ctx, identifiers)
	}
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetMany)
	if len(identifiers) == 0 {
		return nil, nil, nil
	}
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	results := make([]PT, 0, len(identifiers))
	misses := make([]int, 0)
	var notFound int
	for idx, id := range identifiers {
		obj, found := c.cache[id]
		if !found {
			notFound++
			misses = append(misses, idx)
			continue
		}
		if !c.isReadAllowed(ctx, obj) {
			misses = append(misses, idx)
			continue
		}
		results = append(results, obj.CloneVT())
	}
	if notFound > 0 {
		cacheMissTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "GetMany"}).Add(float64(notFound))
	}
	cacheHitTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "GetMany"}).Add(float64(len(results)))
	return results, misses, nil
}

// WalkByQuery iterates over all the objects scoped by the query applies the closure.
func (c *cachedStore[T, PT]) WalkByQuery(ctx context.Context, query *v1.Query, fn func(obj PT) error) error {
	defer c.setCacheOperationDurationTime(time.Now(), ops.WalkByQuery)
	if checkScopeQueries(ctx, query) {
		return c.Walk(ctx, fn)
	}
	cacheBypassTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "WalkByQuery"}).Inc()
	return c.underlyingStore.WalkByQuery(ctx, query, fn)
}

// Walk iterates over all the objects in the store and applies the closure.
func (c *cachedStore[T, PT]) Walk(ctx context.Context, fn func(obj PT) error) error {
	if !c.useCache(ctx) {
		return c.underlyingStore.Walk(ctx, fn)
	}
	objects := concurrency.WithRLock1(&c.cacheLock, func() []PT {
		return slices.AppendSeq(make([]PT, 0, len(c.cache)), maps.Values(c.cache))
	})
	for _, obj := range objects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.isReadAllowed(ctx, obj) {
			if err := fn(obj.CloneVT()); err != nil {
				return err
			}
		}
	}
	return nil
}

// GetAllForSAC bypasses SAC filtering to build access scopes. Returned objects
// are not cloned and must not be modified by callers.
func (c *cachedStore[T, PT]) GetAllForSAC(ctx context.Context) ([]PT, error) {
	if !c.useCache(ctx) {
		return c.underlyingStore.GetAllForSAC(ctx)
	}
	c.cacheLock.RLock()
	defer c.cacheLock.RUnlock()
	return slices.AppendSeq(make([]PT, 0, len(c.cache)), maps.Values(c.cache)), nil
}

// GetByQueryFn iterates over the objects from the store matching the query.
func (c *cachedStore[T, PT]) GetByQueryFn(ctx context.Context, query *v1.Query, fn func(obj PT) error) error {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetByQuery)
	if checkScopeQueries(ctx, query) {
		return c.Walk(ctx, fn)
	}
	cacheBypassTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "GetByQueryFn"}).Inc()
	return c.underlyingStore.GetByQueryFn(ctx, query, fn)
}

// GetByQuery returns the objects from the store matching the query.
func (c *cachedStore[T, PT]) GetByQuery(ctx context.Context, query *v1.Query) ([]*T, error) {
	defer c.setCacheOperationDurationTime(time.Now(), ops.GetByQuery)
	if checkScopeQueries(ctx, query) {
		var result []*T
		err := c.Walk(ctx, func(obj PT) error {
			result = append(result, obj)
			return nil
		})
		if err != nil {
			return nil, err
		}
		return result, err
	}
	cacheBypassTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "GetByQuery"}).Inc()
	return c.underlyingStore.GetByQuery(ctx, query)
}

// DeleteByQuery removes the objects from the store based on the passed query.
func (c *cachedStore[T, PT]) DeleteByQuery(ctx context.Context, query *v1.Query) error {
	_, err := c.deleteByQueryWithIDs(ctx, query)
	return err
}

// DeleteByQueryWithIDs removes the objects from the store based on the passed query returning deleted IDs.
func (c *cachedStore[T, PT]) DeleteByQueryWithIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	return c.deleteByQueryWithIDs(ctx, query)
}

// deleteByQueryWithIDs removes the objects from the store based on the passed query returning deleted IDs.
func (c *cachedStore[T, PT]) deleteByQueryWithIDs(ctx context.Context, query *v1.Query) ([]string, error) {
	defer c.beginMutation(ctx)()
	identifiersToRemove, err := c.underlyingStore.DeleteByQueryWithIDs(ctx, query)
	if err != nil {
		c.mutationFailed()
		return nil, err
	}
	if postgres.HasTxInContext(ctx) {
		return identifiersToRemove, nil
	}
	defer c.signalObservers()
	defer c.setCacheOperationDurationTime(time.Now(), ops.Remove)
	c.cacheLock.Lock()
	defer c.cacheLock.Unlock()
	for _, id := range identifiersToRemove {
		c.removeFromCacheNoLock(id)
	}
	c.setCacheEntriesGauge()
	return identifiersToRemove, nil
}

// GetIDs returns all the IDs for the store.
func (c *cachedStore[T, PT]) GetIDs(ctx context.Context) ([]string, error) {
	if !c.useCache(ctx) {
		return c.underlyingStore.GetIDs(ctx)
	}
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
	if checkScopeQueries(ctx, query) {
		return c.GetIDs(ctx)
	}
	cacheBypassTotal.With(prometheus.Labels{"Type": c.schema.TypeName, "Operation": "GetIDsByQuery"}).Inc()
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

func checkScopeQueries(ctx context.Context, query *v1.Query) bool {
	scopeQuery, err := scoped.GetQueryForAllScopes(ctx)
	if err != nil {
		return false
	}

	if scopeQuery == nil && (query == nil || query.EqualVT(search.EmptyQuery())) {
		return true
	}

	return false
}

func (c *cachedStore[T, PT]) isReadAllowed(ctx context.Context, obj PT) bool {
	return c.isActionAllowed(ctx, storage.Access_READ_ACCESS, obj)
}

func (c *cachedStore[T, PT]) isWriteAllowed(ctx context.Context, obj PT) bool {
	return c.isActionAllowed(ctx, storage.Access_READ_WRITE_ACCESS, obj)
}

func (c *cachedStore[T, PT]) isActionAllowed(ctx context.Context, action storage.Access, obj PT) bool {
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

func (c *cachedStore[T, PT]) populateCache() error {
	timer := prometheus.NewTimer(cachePopulationDuration.WithLabelValues(c.schema.TypeName))
	defer timer.ObserveDuration()
	return c.refreshCache(context.Background(), nil, true)
}

func (c *cachedStore[T, PT]) setCacheEntriesGauge() {
	cacheEntries.WithLabelValues(c.schema.TypeName).Set(float64(len(c.cache)))
}

func (c *cachedStore[T, PT]) addToCacheNoLock(obj PT) {
	if c.registration != nil && !c.trusted {
		return
	}
	c.cache[c.pkGetter(obj)] = obj.CloneVT()
}
