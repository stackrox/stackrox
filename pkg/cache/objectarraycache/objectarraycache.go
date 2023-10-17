package objectarraycache

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

// RefreshFunction is a function that retrieves an array of objects.
type RefreshFunction[T any] func(ctx context.Context) ([]*T, error)

// ObjectArrayCache is a cache for an array of objects.
// The object array in the cache instance are valid for a pre-defined period,
// and are retrieved from the source when they are not valid anymore.
type ObjectArrayCache[T any] struct {
	mutex          sync.RWMutex
	lastRefreshed  time.Time
	validityPeriod time.Duration
	objectCache    []*T
	refreshFn      RefreshFunction[T]
}

// NewObjectArrayCache returns a cache for an object array where the cached
// objects are retrieved using refreshFn and are valid for validityPeriod.
func NewObjectArrayCache[T any](validityPeriod time.Duration, refreshFn RefreshFunction[T]) *ObjectArrayCache[T] {
	return &ObjectArrayCache[T]{
		validityPeriod: validityPeriod,
		refreshFn:      refreshFn,
	}
}

func (c *ObjectArrayCache[T]) GetObjects(ctx context.Context) ([]*T, error) {
	objects, valid := c.getObjectsFromCache()
	if valid {
		return objects, nil
	}

	objects, err := c.refreshFn(ctx)
	if err != nil {
		return nil, err
	}

	c.refreshCache(objects)
	return objects, nil
}

func (c *ObjectArrayCache[T]) getObjectsFromCache() ([]*T, bool) {
	now := time.Now()

	c.mutex.RLock()
	defer c.mutex.RUnlock()

	if now.After(c.lastRefreshed.Add(c.validityPeriod)) {
		// Cache expired, notify that the cache should be refreshed
		return nil, false
	}

	res := make([]*T, 0, len(c.objectCache))
	res = append(res, c.objectCache...)
	return res, true
}

func (c *ObjectArrayCache[T]) refreshCache(objects []*T) {
	refreshTime := time.Now()

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if refreshTime.Before(c.lastRefreshed.Add(c.validityPeriod)) {
		// The cache was refreshed in the meantime, skip cache refresh
		return
	}
	c.lastRefreshed = refreshTime
	if len(c.objectCache) == len(objects) {
		// Save the re-allocation of the object array
		for i := 0; i < len(c.objectCache); i++ {
			c.objectCache[i] = objects[i]
		}
	} else {
		c.objectCache = make([]*T, 0, len(objects))
		for _, obj := range objects {
			c.objectCache = append(c.objectCache, obj)
		}
	}
}

func (c *ObjectArrayCache[T]) Refresh(ctx context.Context) error {
	objects, err := c.refreshFn(ctx)
	if err != nil {
		return err
	}
	c.refreshCache(objects)
	return nil
}
