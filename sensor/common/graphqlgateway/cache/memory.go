package cache

import (
	"context"
	"sync"
	"time"

	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// cacheEntry represents a single entry in the cache with expiration time.
type cacheEntry struct {
	token     string
	expiresAt time.Time
}

// isExpired checks if the entry has expired.
func (e *cacheEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

// MemoryCache is an in-memory implementation of TokenCache with TTL support.
// It uses a background goroutine to periodically clean up expired entries.
type MemoryCache struct {
	data      sync.Map
	stopChan  chan struct{}
	closeOnce sync.Once

	// Metrics
	hits   *int64
	misses *int64
	mu     sync.RWMutex
}

// NewMemoryCache creates a new in-memory token cache with background cleanup.
// The cleanupInterval determines how often expired entries are removed.
func NewMemoryCache(cleanupInterval time.Duration) *MemoryCache {
	c := &MemoryCache{
		stopChan: make(chan struct{}),
		hits:     new(int64),
		misses:   new(int64),
	}

	// Start background cleanup goroutine
	go c.cleanupLoop(cleanupInterval)

	return c
}

// Get retrieves a cached token by key.
// Returns (token, true) if found and not expired.
// Returns ("", false) if not found or expired.
func (c *MemoryCache) Get(ctx context.Context, key CacheKey) (string, bool) {
	val, ok := c.data.Load(key.Key())
	if !ok {
		c.incrementMisses()
		return "", false
	}

	entry := val.(*cacheEntry)
	if entry.isExpired() {
		// Entry expired, remove it and return miss
		c.data.Delete(key.Key())
		c.incrementMisses()
		return "", false
	}

	c.incrementHits()
	return entry.token, true
}

// Set stores a token in the cache with the given TTL.
// If an entry already exists for the key, it will be replaced.
func (c *MemoryCache) Set(ctx context.Context, key CacheKey, token string, ttl time.Duration) {
	entry := &cacheEntry{
		token:     token,
		expiresAt: time.Now().Add(ttl),
	}
	c.data.Store(key.Key(), entry)

	log.Debugw("Token cached",
		logging.String("key", key.String()),
		logging.String("ttl", ttl.String()),
	)
}

// Invalidate removes a specific entry from the cache.
func (c *MemoryCache) Invalidate(key CacheKey) {
	c.data.Delete(key.Key())
	log.Debugw("Cache entry invalidated", logging.String("key", key.String()))
}

// Clear removes all entries from the cache.
func (c *MemoryCache) Clear() {
	c.data.Range(func(key, value interface{}) bool {
		c.data.Delete(key)
		return true
	})
	log.Info("Cache cleared")
}

// Size returns the current number of entries in the cache.
func (c *MemoryCache) Size() int {
	count := 0
	c.data.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Close stops the background cleanup goroutine.
// This should be called when the cache is no longer needed.
func (c *MemoryCache) Close() {
	c.closeOnce.Do(func() {
		close(c.stopChan)
	})
}

// cleanupLoop runs in the background and periodically removes expired entries.
func (c *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			return
		}
	}
}

// cleanup removes all expired entries from the cache.
func (c *MemoryCache) cleanup() {
	var expiredCount int

	c.data.Range(func(key, value interface{}) bool {
		entry := value.(*cacheEntry)
		if entry.isExpired() {
			c.data.Delete(key)
			expiredCount++
		}
		return true
	})

	if expiredCount > 0 {
		log.Debugw("Cleaned up expired cache entries", logging.Int("count", expiredCount))
	}
}

// GetHits returns the total number of cache hits.
func (c *MemoryCache) GetHits() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c.hits
}

// GetMisses returns the total number of cache misses.
func (c *MemoryCache) GetMisses() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return *c.misses
}

// GetHitRate returns the cache hit rate as a percentage (0-100).
// Returns 0 if there have been no requests.
func (c *MemoryCache) GetHitRate() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	hits := *c.hits
	misses := *c.misses
	total := hits + misses

	if total == 0 {
		return 0
	}

	return float64(hits) / float64(total) * 100
}

func (c *MemoryCache) incrementHits() {
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.hits++
}

func (c *MemoryCache) incrementMisses() {
	c.mu.Lock()
	defer c.mu.Unlock()
	*c.misses++
}
