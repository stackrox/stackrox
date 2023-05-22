package lru

/*
This file is a temporary copy of the above referenced source. It is meant to disappear once the pending pull request
to add expiring cache is merged ( https://github.com/hashicorp/golang-lru/pull/116 ).

The main type name was changed from Cache to ExpirableCache, and the underlying LRU implementation is the expirable one.
*/

import (
	"sync"
	"time"
)

const (
	// DefaultEvictedBufferSize defines the default buffer size to store evicted key/val
	DefaultEvictedBufferSize = 16
)

// ExpirableCache is a thread-safe fixed size LRU cache with fixed item time to live.
type ExpirableCache[K comparable, V any] struct {
	lru         *expirableLRU[K, V]
	evictedKeys []K
	evictedVals []V
	onEvictedCB func(k K, v V)
	lock        sync.RWMutex
}

// NewExpirableWithEvict constructs a fixed size cache with the given eviction callback and item ttl.
func NewExpirableWithEvict[K comparable, V any](size int, ttl time.Duration, onEvicted func(key K, value V)) (c *ExpirableCache[K, V], err error) {
	// create a cache with default settings
	c = &ExpirableCache[K, V]{
		onEvictedCB: onEvicted,
	}
	onExpired := onEvicted
	if onEvicted != nil {
		c.initEvictBuffers()
		onEvicted = c.onEvicted
	}
	c.lru = newExpirableLRU(size, onEvicted, onExpired, ttl)
	return
}

func (c *ExpirableCache[K, V]) initEvictBuffers() {
	c.evictedKeys = make([]K, 0, DefaultEvictedBufferSize)
	c.evictedVals = make([]V, 0, DefaultEvictedBufferSize)
}

// onEvicted saves evicted key/val and sent in externally registered callback
// outside of critical section
func (c *ExpirableCache[K, V]) onEvicted(k K, v V) {
	c.evictedKeys = append(c.evictedKeys, k)
	c.evictedVals = append(c.evictedVals, v)
}

// Purge is used to completely clear the cache.
func (c *ExpirableCache[K, V]) Purge() {
	var ks []K
	var vs []V
	c.lock.Lock()
	c.lru.purge()
	if c.onEvictedCB != nil && len(c.evictedKeys) > 0 {
		ks, vs = c.evictedKeys, c.evictedVals
		c.initEvictBuffers()
	}
	c.lock.Unlock()
	// invoke callback outside of critical section
	if c.onEvictedCB != nil {
		for i := 0; i < len(ks); i++ {
			c.onEvictedCB(ks[i], vs[i])
		}
	}
}

// Add adds a value to the cache. Returns true if an eviction occurred.
func (c *ExpirableCache[K, V]) Add(key K, value V) (evicted bool) {
	var k K
	var v V
	c.lock.Lock()
	evicted = c.lru.add(key, value)
	if c.onEvictedCB != nil && evicted {
		k, v = c.evictedKeys[0], c.evictedVals[0]
		c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted {
		c.onEvictedCB(k, v)
	}
	return
}

// Get looks up a key's value from the cache.
func (c *ExpirableCache[K, V]) Get(key K) (value V, ok bool) {
	c.lock.Lock()
	value, ok = c.lru.get(key)
	c.lock.Unlock()
	return value, ok
}

// Contains checks if a key is in the cache, without updating the
// recent-ness or or deleting it for being stale.
func (c *ExpirableCache[K, V]) Contains(key K) bool {
	c.lock.RLock()
	containKey := c.lru.contains(key)
	c.lock.RUnlock()
	return containKey
}

// Peek returns the key value (or undefined if not found) without updating
// the "recently used"-ness of the key
func (c *ExpirableCache[K, V]) Peek(key K) (value V, ok bool) {
	c.lock.RLock()
	value, ok = c.lru.peek(key)
	c.lock.RUnlock()
	return value, ok
}

// ContainsOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *ExpirableCache[K, V]) ContainsOrAdd(key K, value V) (ok, evicted bool) {
	var k K
	var v V
	c.lock.Lock()
	if c.lru.contains(key) {
		c.lock.Unlock()
		return true, false
	}
	evicted = c.lru.add(k, v)
	if c.onEvictedCB != nil && evicted {
		k, v = c.evictedKeys[0], c.evictedVals[0]
		c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted {
		c.onEvictedCB(k, v)
	}
	return false, evicted
}

// PeekOrAdd checks if a key is in the cache without updating the
// recent-ness or deleting it for being stale, and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *ExpirableCache[K, V]) PeekOrAdd(key K, value V) (previous V, ok, evicted bool) {
	var k K
	var v V
	c.lock.Lock()
	previous, ok = c.lru.peek(key)
	if ok {
		c.lock.Unlock()
		return previous, true, false
	}
	evicted = c.lru.add(key, value)
	if c.onEvictedCB != nil && evicted {
		k, v = c.evictedKeys[0], c.evictedVals[0]
		c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted {
		c.onEvictedCB(k, v)
	}
	return
}

// Remove removes the provided key from the cache.
func (c *ExpirableCache[K, V]) Remove(key K) (present bool) {
	var k K
	var v V
	c.lock.Lock()
	present = c.lru.remove(key)
	if c.onEvictedCB != nil && present {
		k, v = c.evictedKeys[0], c.evictedVals[0]
		c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && present {
		c.onEvictedCB(k, v)
	}
	return
}

// Resize changes the cache size.
func (c *ExpirableCache[K, V]) Resize(size int) (evicted int) {
	var ks []K
	var vs []V
	c.lock.Lock()
	evicted = c.lru.resize(size)
	if c.onEvictedCB != nil && evicted > 0 {
		ks, vs = c.evictedKeys, c.evictedVals
		c.initEvictBuffers()
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && evicted > 0 {
		for i := 0; i < len(ks); i++ {
			c.onEvictedCB(ks[i], vs[i])
		}
	}
	return evicted
}

// RemoveOldest removes the oldest item from the cache.
func (c *ExpirableCache[K, V]) RemoveOldest() (key K, value V, ok bool) {
	var k K
	var v V
	c.lock.Lock()
	key, value, ok = c.lru.removeOldest()
	if c.onEvictedCB != nil && ok {
		k, v = c.evictedKeys[0], c.evictedVals[0]
		c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	}
	c.lock.Unlock()
	if c.onEvictedCB != nil && ok {
		c.onEvictedCB(k, v)
	}
	return
}

// GetOldest returns the oldest entry
func (c *ExpirableCache[K, V]) GetOldest() (key K, value V, ok bool) {
	c.lock.RLock()
	key, value, ok = c.lru.getOldest()
	c.lock.RUnlock()
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *ExpirableCache[K, V]) Keys() []K {
	c.lock.RLock()
	keys := c.lru.keys()
	c.lock.RUnlock()
	return keys
}

// Len returns the number of items in the cache.
func (c *ExpirableCache[K, V]) Len() int {
	c.lock.RLock()
	length := c.lru.len()
	c.lock.RUnlock()
	return length
}
