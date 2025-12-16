package cache

import (
	"context"
	"time"
)

// TokenCache provides an interface for caching scoped tokens.
// Implementations must be thread-safe.
type TokenCache interface {
	// Get retrieves a cached token by key.
	// Returns (token, true) if found and not expired.
	// Returns ("", false) if not found or expired.
	Get(ctx context.Context, key CacheKey) (string, bool)

	// Set stores a token in the cache with the given TTL.
	// If an entry already exists for the key, it will be replaced.
	Set(ctx context.Context, key CacheKey, token string, ttl time.Duration)

	// Invalidate removes a specific entry from the cache.
	Invalidate(key CacheKey)

	// Clear removes all entries from the cache.
	Clear()

	// Size returns the current number of entries in the cache.
	Size() int
}
