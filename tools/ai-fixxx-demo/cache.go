package demo

import (
	"fmt"
	"sync"
	"time"
)

// Cache is a simple in-memory key-value store with TTL support.
type Cache struct {
	mu      sync.RWMutex
	items   map[string]*cacheItem
	maxSize int
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewCache creates a new cache with the specifed maximum size.
func NewCache(maxSize int) *Cache {
	return &Cache{
		items:   make(map[string]*cacheItem),
		maxSize: maxSize,
	}
}

// Get retreives a value from the cache.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}
	if time.Now().After(item.expiresAt) {
		return nil, false
	}
	return item.value, true
}

// Set adds or updates a value in the cahce with a TTL.
func (c *Cache) Set(key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxSize {
		if _, exists := c.items[key]; !exists {
			return fmt.Errorf("cache is full (max size: %d)", c.maxSize)
		}
	}

	c.items[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

// Delete removes an item from the cache.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Size returns the number of items currently in the cache.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Cleanup removes all expired entries from the cache.
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	now := time.Now()
	for key, item := range c.items {
		if now.After(item.expiresAt) {
			delete(c.items, key)
			removed++
		}
	}
	return removed
}

// Keys returns all non-expired keys in the cache.
func (c *Cache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]string, 0)
	now := time.Now()
	for key, item := range c.items {
		if !now.After(item.expiresAt) {
			keys = append(keys, key)
		}
	}
	return keys
}
