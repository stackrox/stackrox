package lru

import (
	"testing"
	"time"
)

// TestCache is the interface to a fixed-size LRU cache that allows management of potential item expiration
// within the cache.
type TestCache[K comparable, V any] interface {
	Cache[K, V]

	// ExpireItem changes the metadata associated to the input key to mark it as candidate for expiration.
	ExpireItem(t *testing.T, key K)

	// TriggerExpiration makes sure the expired item cleanup loop is triggered on all items present in the cache.
	TriggerExpiration(t *testing.T)
}

type testExpirableLRU[K comparable, V any] struct {
	underlying *expirableLRU[K, V]
}

// NewTestExpirableLRU returns a new thread-safe cache with expirable entries.
// It exposes on top some management functions to test expiration without having to rely on time.Sleep.
func NewTestExpirableLRU[K comparable, V any](
	_ *testing.T,
	size int,
	onEvict EvictCallback[K, V],
	ttl time.Duration,
) TestCache[K, V] {
	if ttl <= 0 {
		ttl = noEvictionTTL
	}
	testCache := &testExpirableLRU[K, V]{
		underlying: &expirableLRU[K, V]{},
	}
	initExpirableLRU(testCache.underlying, size, onEvict, ttl)
	return testCache
}

// Purge clears the cache completely.
// onEvict is called for each evicted key.
func (c *testExpirableLRU[K, V]) Purge() {
	c.underlying.Purge()
}

// Add adds a value to the cache. Returns true if an eviction occurred.
// Returns false if there was no eviction: the item was already in the cache,
// or the size was not exceeded.
func (c *testExpirableLRU[K, V]) Add(key K, value V) bool {
	return c.underlying.Add(key, value)
}

// Get looks up a key's value from the cache.
func (c *testExpirableLRU[K, V]) Get(key K) (value V, ok bool) {
	return c.underlying.Get(key)
}

// Contains checks if a key is in the cache, without updating the recent-ness or deleting it for being stale.
func (c *testExpirableLRU[K, V]) Contains(key K) bool {
	return c.underlying.Contains(key)
}

// Peek returns the key value (or undefined if not found) without updating the "recentlyused"-ness of the key.
func (c *testExpirableLRU[K, V]) Peek(key K) (value V, ok bool) {
	return c.underlying.Peek(key)
}

// ContainsOrAdd checks if a key is in the cache without updating the recent-ness or deleting if for being stale,
// and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *testExpirableLRU[K, V]) ContainsOrAdd(key K, value V) (ok, evicted bool) {
	return c.underlying.ContainsOrAdd(key, value)
}

// PeekOrAdd checks if a key is in the cache without updating the recent-ness or deleting it for being stale,
// and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *testExpirableLRU[K, V]) PeekOrAdd(key K, value V) (previous V, ok bool, evicted bool) {
	return c.underlying.PeekOrAdd(key, value)
}

// Remove removes the provided key from the cache.
func (c *testExpirableLRU[K, V]) Remove(key K) (present bool) {
	return c.underlying.Remove(key)
}

// Resize changes the cache size.
func (c *testExpirableLRU[K, V]) Resize(size int) (evicted int) {
	return c.underlying.Resize(size)
}

// RemoveOldest removes the oldest item from the cache.
func (c *testExpirableLRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	return c.underlying.RemoveOldest()
}

// GetOldest returns the oldest entry.
func (c *testExpirableLRU[K, V]) GetOldest() (key K, value V, ok bool) {
	return c.underlying.GetOldest()
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *testExpirableLRU[K, V]) Keys() []K {
	return c.underlying.Keys()
}

// Values returns a slice of the values in the cache, from oldest to newest.
func (c *testExpirableLRU[K, V]) Values() []V {
	return c.underlying.Values()
}

// Len returns the number of items in the cache.
func (c *testExpirableLRU[K, V]) Len() int {
	return c.underlying.Len()
}

// Close destroys internal cache resources. To clean up the cache, run Purge() before Close().
func (c *testExpirableLRU[K, V]) Close() {
	c.underlying.Close()
}

// ExpireItem changes the metadata associated to the input key to mark it as candidate for expiration.
func (c *testExpirableLRU[K, V]) ExpireItem(_ *testing.T, key K) {
	c.underlying.items[key].expiresAt = time.Now().Add(-5 * time.Millisecond)
}

// TriggerExpiration makes sure the expired item cleanup loop is triggered on all items present in the cache.
func (c *testExpirableLRU[K, V]) TriggerExpiration(_ *testing.T) {
	for i := 0; i < 2*numBuckets; i++ {
		c.underlying.deleteExpired()
	}
}
