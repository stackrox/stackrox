package simplecache

import "github.com/stackrox/rox/pkg/sync"

// Cache offers a generic, threadsafe interface for a map based cache
type Cache interface {
	Add(key, value any)
	Get(key any) (any, bool)
	Remove(key any) (any, bool)
	Size() int
	Keys() []any
}

// New creates a new simple map backed cache
func New() Cache {
	return &cacheImpl{
		cache: make(map[any]any),
	}
}

type cacheImpl struct {
	lock  sync.RWMutex
	cache map[any]any
}

func (c *cacheImpl) Add(k, v any) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.cache[k] = v
}

func (c *cacheImpl) Get(k any) (any, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	val, ok := c.cache[k]
	if !ok {
		return nil, false
	}
	return val, true
}

func (c *cacheImpl) Remove(k any) (any, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	v, ok := c.cache[k]
	if !ok {
		return nil, false
	}
	delete(c.cache, k)
	return v, true
}

func (c *cacheImpl) Keys() []any {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var keys = make([]any, 0, len(c.cache))
	for k := range c.cache {
		keys = append(keys, k)
	}
	return keys
}

func (c *cacheImpl) Size() int {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return len(c.cache)
}
