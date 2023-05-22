// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

/*
source: https://github.com/hashicorp/golang-lru/blob/8d9a62dcf60cd87ed918b57afad8a001d25db3de/simplelru/lru_interface.go
This file is a temporary copy of the above referenced source. It is meant to disappear once the pending pull request
to add expiring cache is merged ( https://github.com/hashicorp/golang-lru/pull/116 ).

The interface methods were made private as these are for a package-internal object, and the interface is only used for
testing purposes.
*/

package lru

// lruCache is the interface for simple LRU cache.
type lruCache[K comparable, V any] interface {
	// Adds a value to the cache, returns true if an eviction occurred and
	// updates the "recently used"-ness of the key.
	add(key K, value V) bool

	// Returns key's value from the cache and
	// updates the "recently used"-ness of the key. #value, isFound
	get(key K) (value V, ok bool)

	// Checks if a key exists in cache without updating the recent-ness.
	contains(key K) (ok bool)

	// Returns key's value without updating the "recently used"-ness of the key.
	peek(key K) (value V, ok bool)

	// Removes a key from the cache.
	remove(key K) bool

	// Removes the oldest entry from cache.
	removeOldest() (K, V, bool)

	// Returns the oldest entry from the cache. #key, value, isFound
	getOldest() (K, V, bool)

	// Returns a slice of the keys in the cache, from oldest to newest.
	keys() []K

	// Returns the number of items in the cache.
	len() int

	// Clears all cache entries.
	purge()

	// Resizes cache, returning number evicted
	resize(int) int
}
