package lru

// LRU is the interface for a key-value Cache with accessor to the least
// recently used item.
//
//go:generate mockgen-wrapper
type LRU[K comparable, V any] interface {
	// Purge clears the cache completely.
	Purge()
	// Add adds a value to the cache. Returns true if an eviction occurred.
	// Returns false if there was no eviction: the item was already in the cache,
	// or the size was not exceeded.
	Add(key K, value V) (evicted bool)
	// Remove removes the provided key from the cache, returning if the
	// key was contained.
	Remove(key K) bool
	// Get looks up a key's value from the cache.
	Get(key K) (value V, ok bool)
	// Peek returns the key value (or undefined if not found) without updating
	// the "recently used"-ness of the key.
	Peek(key K) (value V, ok bool)
	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K
	/*
		// Here are the interface functions exposed by hashicorp golang-lru types,
		// which are not used in our software (yet).

		// Contains checks if a key is in the cache, without updating the recent-ness
		// or deleting it for being stale.
		Contains(key K) (ok bool)
		// RemoveOldest removes the oldest item from the cache.
		RemoveOldest() (key K, value V, ok bool)
		// GetOldest returns the oldest entry.
		GetOldest() (key K, value V, ok bool)
		// Values returns a slice of the values in the cache, from oldest to newest.
		Values() []V
		// Len returns the number of items in the cache.
		Len() int
		// Resize changes the cache size. Size of 0 means unlimited.
		Resize(size int) (evicted int)
	*/
}
