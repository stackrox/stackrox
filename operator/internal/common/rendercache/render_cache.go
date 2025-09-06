package rendercache

import (
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/types"
)

// RenderCache is a thread-safe map that stores shared data between Extensions and Renderers,
// accessible by the reconciled objects UID
type RenderCache struct {
	mu   sync.RWMutex
	Data map[types.UID]RenderData
}

// RenderData contains data that can be shared between Extensions and the Renderer
type RenderData struct {
	CAHash string
	// other data can be added in the future
}

// NewRenderCache creates an initialized cache.
func NewRenderCache() *RenderCache {
	return &RenderCache{
		Data: make(map[types.UID]RenderData),
	}
}

// Set stores the RenderData for a given object's UID.
func (c *RenderCache) Set(objUID types.UID, data RenderData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data[objUID] = data
}

// Get retrieves the RenderData for a given object's UID.
func (c *RenderCache) Get(objUID types.UID) (RenderData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, found := c.Data[objUID]
	return data, found
}

// Delete removes the RenderData for a given object's UID.
func (c *RenderCache) Delete(objUID types.UID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Data, objUID)
}

// Clear removes all data from the cache.
func (c *RenderCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data = make(map[types.UID]RenderData)
}
