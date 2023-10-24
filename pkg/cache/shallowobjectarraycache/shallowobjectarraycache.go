package objectarraycache

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

// RefreshFunction is a function that retrieves an array of objects.
type RefreshFunction[T any] func(ctx context.Context) ([]*T, error)

// ShallowObjectArrayCache is a cache for an array of objects.
// The object array in the cache instance are valid for a pre-defined period,
// and are retrieved from the source when they are not valid anymore.
type ShallowObjectArrayCache[T any] struct {
	mutex              sync.RWMutex
	lastRefreshTrigger time.Time
	lastRefreshed      time.Time
	validityPeriod     time.Duration
	objectCache        []*T
	refreshFn          RefreshFunction[T]
	refreshCtx         context.Context
}

// NewShallowObjectArrayCache returns a cache for an object array where the cached
// objects are retrieved using refreshFn and are valid for validityPeriod.
func NewShallowObjectArrayCache[T any](
	refreshCtx context.Context,
	validityPeriod time.Duration,
	refreshFn RefreshFunction[T],
) *ShallowObjectArrayCache[T] {
	return &ShallowObjectArrayCache[T]{
		validityPeriod: validityPeriod,
		refreshFn:      refreshFn,
		refreshCtx:     refreshCtx,
	}
}

// GetObjects retrieves the array of objects, either from cache (when not
// expired), or using the refresh function (when the cache is expired).
// The returned objects are not copied, and could be subject to concurrent
// modifications. If the functional flow modifies the retrieved objects,
// concurrency has to be handled at object level.
func (c *ShallowObjectArrayCache[T]) GetObjects() []*T {
	objects, valid := c.getObjectsFromCache()

	if !valid {
		// Cache is not fresh anymore, refresh it in the background
		go c.doBackgroundRefresh()
		if len(objects) == 0 {
			objects, _ = c.refreshFn(c.refreshCtx)
		}
	}

	return objects
}

func (c *ShallowObjectArrayCache[T]) getObjectsFromCache() ([]*T, bool) {
	now := time.Now()
	valid := true

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if now.After(c.lastRefreshed.Add(c.validityPeriod)) {
		// Cache expired, notify that the cache should be refreshed
		valid = false
	}

	res := make([]*T, 0, len(c.objectCache))
	res = append(res, c.objectCache...)
	return res, valid
}

func (c *ShallowObjectArrayCache[T]) doBackgroundRefresh() {
	shouldRefresh := c.triggerBackgroundRefresh()
	if !shouldRefresh {
		return
	}
	_ = c.Refresh(c.refreshCtx)
}

func (c *ShallowObjectArrayCache[T]) triggerBackgroundRefresh() bool {
	now := time.Now()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if now.Before(c.lastRefreshTrigger.Add(c.validityPeriod)) {
		// Another execution picked the refresh request, stop here.
		return false
	}

	c.lastRefreshTrigger = now

	return true
}

func (c *ShallowObjectArrayCache[T]) refreshCache(objects []*T) {
	refreshTime := time.Now()

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if refreshTime.Before(c.lastRefreshed.Add(c.validityPeriod)) {
		// The cache was refreshed in the meantime, skip cache refresh
		return
	}
	c.lastRefreshTrigger = refreshTime
	c.lastRefreshed = refreshTime
	if len(c.objectCache) == len(objects) {
		// Save the re-allocation of the object array
		c.objectCache = c.objectCache[:0]
	} else {
		c.objectCache = make([]*T, 0, len(objects))
	}
	c.objectCache = append(c.objectCache, objects...)
}

// Refresh triggers a manual refresh of the cache.
func (c *ShallowObjectArrayCache[T]) Refresh(ctx context.Context) error {
	objects, err := c.refreshFn(ctx)
	if err != nil {
		return err
	}
	c.refreshCache(objects)
	return nil
}
