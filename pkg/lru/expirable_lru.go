package lru

/*
source: https://github.com/hashicorp/golang-lru/pull/116/files#diff-be4b9429f63e821595d57501426a7f91505ab8f2df186c5f5e3b0af74e1a5dfc
This file is a temporary copy of the above referenced source with some adjustments. It is meant to disappear once the pending pull request
to add expiring cache is merged ( https://github.com/hashicorp/golang-lru/pull/116 ).
*/

import (
	"time"

	"github.com/stackrox/rox/pkg/sync"
)

// EvictCallback is used to get a callback when a cache entry is evicted.
type EvictCallback[K comparable, V any] func(key K, value V)

// expirableLRU implements a thread-safe LRU with expirable entries.
type expirableLRU[K comparable, V any] struct {
	size      int
	evictList *lruList[K, V]
	items     map[K]*entry[K, V]
	onEvict   EvictCallback[K, V]

	evictedKeys []K
	evictedVals []V

	// expirable options
	mu   sync.Mutex
	ttl  time.Duration
	done chan struct{}

	// buckets for expiration
	buckets []bucket[K, V]
	// uint8 because it's number between 0 and numBuckets
	nextCleanupBucket uint8
}

// bucket is a container for holding entries to be expired
type bucket[K comparable, V any] struct {
	entries map[K]*entry[K, V]
}

const (
	// noEvictionTTL = very long ttl to prevent eviction
	noEvictionTTL = 10 * 365 * 24 * time.Hour

	// because of uint8 usage for nextCleanupBucket, should not exceed 256
	numBuckets = 100

	// defaultEvictBufferSize defines the default buffer size to store evicted key/val
	defaultEvictedBufferSize = 16
)

func initExpirableLRU[K comparable, V any](
	obj *expirableLRU[K, V],
	size int,
	onEvict EvictCallback[K, V],
	ttl time.Duration,
	tickerChannel <-chan time.Time,
) {

	if size < 0 {
		size = 0
	}
	if ttl <= 0 {
		ttl = noEvictionTTL
	}

	obj.ttl = ttl
	obj.size = size
	obj.evictList = newList[K, V]()
	obj.items = make(map[K]*entry[K, V])
	obj.onEvict = onEvict
	obj.done = make(chan struct{}, 1)

	// initialize the buckets
	obj.buckets = make([]bucket[K, V], numBuckets)
	for i := 0; i < numBuckets; i++ {
		obj.buckets[i] = bucket[K, V]{entries: make(map[K]*entry[K, V])}
	}

	// enable deleteExpired() running in separate goroutine for cache with non-zero TTL
	if obj.ttl != noEvictionTTL {
		go func(done <-chan struct{}) {
			for {
				select {
				case <-done:
					return
				case <-tickerChannel:
					obj.deleteExpired()
				}
			}
		}(obj.done)
	}
}

// NewExpirableLRU returns a new thread-safe cache with expirable entries.
//
// Size parameter set to 0 makes cache of unlimited size, e.g. turns LRU mechanism off.
//
// Providing 0 TTL turns expiring off.
//
// Delete expired entries every 1/100th of ttl value.
func NewExpirableLRU[K comparable, V any](
	size int,
	onEvict EvictCallback[K, V],
	ttl time.Duration,
) Cache[K, V] {
	if ttl <= 0 {
		ttl = noEvictionTTL
	}
	ticker := time.NewTicker(ttl / numBuckets)
	res := &expirableLRU[K, V]{}
	initExpirableLRU(res, size, onEvict, ttl, ticker.C)
	return res
}

// Purge clears the cache completely.
// onEvict is called for each evicted key.
func (c *expirableLRU[K, V]) Purge() {
	var ks []K
	var vs []V
	c.mu.Lock()
	for _, e := range c.items {
		c.removeElement(e)
	}
	if c.onEvict != nil {
		ks, vs = c.evictedKeys, c.evictedVals
	}
	c.initEvictBuffers()
	c.evictList.init()
	c.mu.Unlock()
	if c.onEvict != nil {
		for i := 0; i < len(ks); i++ {
			c.onEvict(ks[i], vs[i])
		}
	}
}

// Add adds a value to the cache. Returns true if an eviction occurred.
// Returns false if there was no eviction: the item was already in the cache,
// or the size was not exceeded.
func (c *expirableLRU[K, V]) Add(key K, value V) (evicted bool) {
	var k K
	var v V
	c.mu.Lock()
	if ent, ok := c.items[key]; ok {
		c.evictList.moveToFront(ent)
		c.removeFromBucket(ent) // remove the entry from its current bucket as expiresAt is renewed
		if c.onEvict != nil {
			c.evict(ent)
		}
		ent.value = value
		ent.expiresAt = time.Now().Add(c.ttl)
		c.addToBucket(ent)
		evicted = false
	} else {
		evicted = c.addNewItem(key, value)
	}
	if c.onEvict != nil && len(c.evictedKeys) > 0 {
		k, v = c.getEvictedKeyValuePair()
	}
	c.mu.Unlock()
	if c.onEvict != nil && evicted {
		c.onEvict(k, v)
	}
	return
}

// Get looks up a key's value from the cache.
func (c *expirableLRU[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var ent *entry[K, V]
	if ent, ok = c.items[key]; ok {
		if time.Now().After(ent.expiresAt) {
			return
		}
		c.evictList.moveToFront(ent)
		return ent.value, true
	}
	return
}

// Contains checks if a key is in the cache, without updating the recent-ness or deleting it for being stale.
func (c *expirableLRU[K, V]) Contains(key K) bool {
	c.mu.Lock()
	ent, containsKey := c.items[key]
	if containsKey && time.Now().After(ent.expiresAt) {
		containsKey = false
	}
	c.mu.Unlock()
	return containsKey
}

// Peek returns the key value (or undefined if not found) without updating the "recentlyused"-ness of the key.
func (c *expirableLRU[K, V]) Peek(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ent, ok := c.items[key]; ok && time.Now().Before(ent.expiresAt) {
		return ent.value, true
	}
	return
}

// ContainsOrAdd checks if a key is in the cache without updating the recent-ness or deleting if for being stale,
// and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *expirableLRU[K, V]) ContainsOrAdd(key K, value V) (ok, evicted bool) {
	var k K
	var v V
	c.mu.Lock()
	if ent, ok := c.items[key]; ok && time.Now().Before(ent.expiresAt) {
		c.mu.Unlock()
		return true, false
	}
	evicted = c.addNewItem(key, value)
	if c.onEvict != nil && evicted {
		k, v = c.getEvictedKeyValuePair()
	}
	c.mu.Unlock()
	if c.onEvict != nil && evicted {
		c.onEvict(k, v)
	}
	return false, evicted
}

// PeekOrAdd checks if a key is in the cache without updating the recent-ness or deleting it for being stale,
// and if not, adds the value.
// Returns whether found and whether an eviction occurred.
func (c *expirableLRU[K, V]) PeekOrAdd(key K, value V) (previous V, ok bool, evicted bool) {
	var k K
	var v V
	c.mu.Lock()
	if ent, found := c.items[key]; found && time.Now().Before(ent.expiresAt) {
		previous, ok, evicted = ent.value, true, false
		c.mu.Unlock()
		return
	}
	evicted = c.addNewItem(key, value)
	if c.onEvict != nil && evicted {
		k, v = c.getEvictedKeyValuePair()
	}
	c.mu.Unlock()
	if c.onEvict != nil && evicted {
		c.onEvict(k, v)
	}
	return
}

// Remove removes the provided key from the cache.
func (c *expirableLRU[K, V]) Remove(key K) (present bool) {
	var k K
	var v V
	c.mu.Lock()
	if ent, ok := c.items[key]; ok {
		c.removeElement(ent)
		present = true
	}
	if c.onEvict != nil && present {
		k, v = c.getEvictedKeyValuePair()
	}
	c.mu.Unlock()
	if c.onEvict != nil && present {
		c.onEvict(k, v)
	}
	return
}

// Resize changes the cache size.
func (c *expirableLRU[K, V]) Resize(size int) (evicted int) {
	var ks []K
	var vs []V
	c.mu.Lock()
	if size <= 0 {
		c.size = 0
		evicted = 0
		c.mu.Unlock()
		return
	}
	evicted = c.evictList.length() - size
	if evicted < 0 {
		evicted = 0
	}
	for i := 0; i < evicted; i++ {
		c.removeOldest()
	}
	c.size = size
	if c.onEvict != nil && evicted > 0 {
		ks, vs = c.evictedKeys, c.evictedVals
		c.initEvictBuffers()
	}
	c.mu.Unlock()
	if c.onEvict != nil && evicted > 0 {
		for i := 0; i < len(ks); i++ {
			c.onEvict(ks[i], vs[i])
		}
	}
	return evicted
}

// RemoveOldest removes the oldest item from the cache.
func (c *expirableLRU[K, V]) RemoveOldest() (key K, value V, ok bool) {
	var k K
	var v V
	c.mu.Lock()
	if ent := c.evictList.back(); ent != nil {
		key, value = ent.key, ent.value
		c.removeElement(ent)
		ok = true
	}
	if c.onEvict != nil && ok {
		k, v = c.getEvictedKeyValuePair()
	}
	c.mu.Unlock()
	if c.onEvict != nil && ok {
		c.onEvict(k, v)
	}
	return
}

// GetOldest returns the oldest entry.
func (c *expirableLRU[K, V]) GetOldest() (key K, value V, ok bool) {
	c.mu.Lock()
	if ent := c.evictList.back(); ent != nil {
		key, value, ok = ent.key, ent.value, true
	}
	c.mu.Unlock()
	return
}

// Keys returns a slice of the keys in the cache, from oldest to newest.
func (c *expirableLRU[K, V]) Keys() []K {
	c.mu.Lock()
	keys := make([]K, c.evictList.length())
	i := 0
	for ent := c.evictList.back(); ent != nil; ent = ent.prevEntry() {
		keys[i] = ent.key
		i++
	}
	c.mu.Unlock()
	return keys
}

// Values returns a slice of the values in the cache, from oldest to newest.
func (c *expirableLRU[K, V]) Values() []V {
	c.mu.Lock()
	values := make([]V, c.evictList.length())
	i := 0
	for ent := c.evictList.back(); ent != nil; ent = ent.prevEntry() {
		values[i] = ent.value
		i++
	}
	c.mu.Unlock()
	return values
}

// Len returns the number of items in the cache.
func (c *expirableLRU[K, V]) Len() int {
	c.mu.Lock()
	length := c.evictList.length()
	c.mu.Unlock()
	return length
}

// Close destroys internal cache resources. To clean up the cache, run Purge() before Close().
func (c *expirableLRU[K, V]) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case <-c.done:
		return
	default:
	}
	c.done <- struct{}{}
	close(c.done)
}

func (c *expirableLRU[K, V]) initEvictBuffers() {
	c.evictedKeys = make([]K, 0, defaultEvictedBufferSize)
	c.evictedVals = make([]V, 0, defaultEvictedBufferSize)
}

func (c *expirableLRU[K, V]) evict(e *entry[K, V]) {
	c.evictedKeys = append(c.evictedKeys, e.key)
	c.evictedVals = append(c.evictedVals, e.value)
}

func (c *expirableLRU[K, V]) addNewItem(key K, value V) (evicted bool) {
	ent := c.evictList.pushFrontExpirable(key, value, time.Now().Add(c.ttl))
	c.items[key] = ent
	c.addToBucket(ent)
	evicted = c.size > 0 && c.evictList.length() > c.size
	// Verify size not exceeded
	if evicted {
		c.removeOldest()
	}
	return evicted
}

func (c *expirableLRU[K, V]) getEvictedKeyValuePair() (K, V) {
	key, val := c.evictedKeys[0], c.evictedVals[0]
	c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	return key, val
}

// removeOldest removes the oldest item from the cache. Has to be called with lock!
func (c *expirableLRU[K, V]) removeOldest() {
	if ent := c.evictList.back(); ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache. Has to be called with lock!
func (c *expirableLRU[K, V]) removeElement(e *entry[K, V]) {
	c.evictList.remove(e)
	delete(c.items, e.key)
	c.removeFromBucket(e)
	if c.onEvict != nil {
		c.evict(e)
	}
}

// deleteExpired deletes expired records. Does check for entry.expiresAt as it could be
// TTL/numBuckets in the future.
func (c *expirableLRU[K, V]) deleteExpired() {
	c.mu.Lock()
	keys, vals := c.deleteExpiredNoLock()
	c.mu.Unlock()
	for i := 0; i < len(keys); i++ {
		c.onEvict(keys[i], vals[i])
	}
}

// deleteExpiredNoLock deletes expired records. Does check for entry.expiresAt as it could be
// TTL/numBuckets in the future.
// Has to be called with lock!
func (c *expirableLRU[K, V]) deleteExpiredNoLock() ([]K, []V) {
	bucketIdx := c.nextCleanupBucket
	now := time.Now()
	for _, ent := range c.buckets[bucketIdx].entries {
		if ent.expiresAt.After(now) {
			continue
		}
		c.removeElement(ent)
	}
	c.nextCleanupBucket = (c.nextCleanupBucket + 1) % numBuckets
	expiredKeys, expiredValues := c.evictedKeys, c.evictedVals
	c.evictedKeys, c.evictedVals = c.evictedKeys[:0], c.evictedVals[:0]
	return expiredKeys, expiredValues
}

// removeFromBucket removes the entry from its corresponding bucket. Has to be called with lock!
func (c *expirableLRU[K, V]) removeFromBucket(e *entry[K, V]) {
	delete(c.buckets[e.expireBucket].entries, e.key)
}

// addToBucket adds entry to expire bucket so that it will be cleaned up when the time comes. Has to be called with lock!
func (c *expirableLRU[K, V]) addToBucket(e *entry[K, V]) {
	bucketID := (numBuckets + c.nextCleanupBucket - 1) % numBuckets
	e.expireBucket = bucketID
	c.buckets[bucketID].entries[e.key] = e
}
