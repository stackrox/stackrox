package postgres

import (
	"sync"

	"github.com/stackrox/rox/generated/storage"
)

// nodeCVECache is a thread-safe in-memory cache for NodeCVE objects
type nodeCVECache struct {
	mu    sync.RWMutex
	cache map[string]*storage.NodeCVE
}

// newNodeCVECache creates a new thread-safe NodeCVE cache
func newNodeCVECache() *nodeCVECache {
	return &nodeCVECache{
		cache: make(map[string]*storage.NodeCVE, 256),
	}
}

func (c *nodeCVECache) GetMany(ids []string) (map[string]*storage.NodeCVE, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*storage.NodeCVE, len(ids))
	missing := make([]string, 0)
	for _, id := range ids {
		if cve, exists := c.cache[id]; exists {
			// Return a clone to prevent external modifications
			result[id] = cve.CloneVT()
		} else {
			missing = append(missing, id)
		}
	}
	return result, missing
}

func (c *nodeCVECache) SetMany(cves map[string]*storage.NodeCVE) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, cve := range cves {
		// Store a clone to prevent external modifications
		c.cache[id] = cve.CloneVT()
	}
}

func (c *nodeCVECache) DeleteMany(ids []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, id := range ids {
		delete(c.cache, id)
	}
}
