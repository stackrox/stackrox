package rendercache

import (
	"github.com/stackrox/rox/pkg/sync"
	"k8s.io/apimachinery/pkg/types"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
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

// Set stores the RenderData for a given object.
func (c *RenderCache) Set(obj ctrlClient.Object, data RenderData) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data[obj.GetUID()] = data
}

// Get retrieves the RenderData for a given object.
func (c *RenderCache) Get(obj ctrlClient.Object) (RenderData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	data, found := c.Data[obj.GetUID()]
	return data, found
}

// Delete removes the RenderData for a given object.
func (c *RenderCache) Delete(obj ctrlClient.Object) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Data, obj.GetUID())
}

// Clear removes all data from the cache.
func (c *RenderCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Data = make(map[types.UID]RenderData)
}
