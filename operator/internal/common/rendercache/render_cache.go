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
	data map[types.UID]RenderData
}

// RenderData contains data that can be shared between Extensions and the Renderer
type RenderData struct {
	// CAHash is a hash of the CA used to sign the TLS certificates for the Central / Secured Cluster.
	CAHash string
	// this struct can be extended if needed, new accessors will have to be added to the RenderCache for the new fields
}

// NewRenderCache creates an initialized cache.
func NewRenderCache() *RenderCache {
	return &RenderCache{
		data: make(map[types.UID]RenderData),
	}
}

// SetCAHash stores the CA hash for a given object.
func (c *RenderCache) SetCAHash(obj ctrlClient.Object, caHash string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	uid := obj.GetUID()
	data := c.data[uid]
	data.CAHash = caHash
	c.data[uid] = data
}

// GetCAHash retrieves the CA hash for a given object.
func (c *RenderCache) GetCAHash(obj ctrlClient.Object) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	data, found := c.data[obj.GetUID()]
	if !found {
		return "", false
	}
	return data.CAHash, data.CAHash != ""
}

// Delete removes the RenderData for a given object.
func (c *RenderCache) Delete(obj ctrlClient.Object) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, obj.GetUID())
}

// Clear removes all data from the cache.
func (c *RenderCache) Clear() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[types.UID]RenderData)
}
