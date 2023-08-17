package lru

// Cache is the interface to a fixed size LRU cache.
type Cache[K comparable, V any] interface {
	// Contains checks if a key is in the cache, without updating the recent-ness or deleting it for being stale.
	Contains(key K) bool
	// Get looks up a key's value from the cache.
	Get(key K) (value V, ok bool)
	// GetOldest returns the oldest entry.
	GetOldest() (key K, value V, ok bool)
	// Peek returns the key value (or undefined if not found) without updating the "recentlyused"-ness of the key.
	Peek(key K) (value V, ok bool)

	// Keys returns a slice of the keys in the cache, from oldest to newest.
	Keys() []K
	// Values returns a slice of the values in the cache, from oldest to newest.
	Values() []V
	// Len returns the number of items in the cache.
	Len() int

	// Add adds a value to the cache. Returns true if an eviction occurred.
	Add(key K, value V) (evicted bool)
	// ContainsOrAdd checks if a key is in the cache without updating the recent-ness or deleting if for being stale,
	// and if not, adds the value.
	// Returns whether found and whether an eviction occurred.
	ContainsOrAdd(key K, value V) (ok bool, evicted bool)
	// PeekOrAdd checks if a key is in the cache without updating the recent-ness or deleting it for being stale,
	// and if not, adds the value.
	// Returns whether found and whether an eviction occurred.
	PeekOrAdd(key K, value V) (previous V, ok bool, evicted bool)
	// Purge is used to completely clear the cache.
	Purge()
	// Remove removes the provided key from the cache.
	Remove(key K) (present bool)
	// RemoveOldest removes the oldest item from the cache.
	RemoveOldest() (key K, value V, ok bool)
	// Resize changes the cache size.
	Resize(size int) (evicted int)

	// Close destroys internal cache resources. To clean up the cache, run Purge() before Close().
	Close()
}
