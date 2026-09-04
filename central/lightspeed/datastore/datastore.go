package datastore

import "github.com/stackrox/rox/pkg/sync"

// LightspeedInfo stores per-cluster OpenShift Lightspeed availability data.
type LightspeedInfo struct {
	IsAvailable bool
	Endpoint    string
}

// DataStore stores per-cluster OpenShift Lightspeed availability state in memory.
type DataStore interface {
	Update(clusterID string, info LightspeedInfo)
	Get(clusterID string) (LightspeedInfo, bool)
	Remove(clusterID string)
	GetAll() map[string]LightspeedInfo
}

// New returns a new in-memory DataStore.
func New() DataStore {
	return &datastoreImpl{
		store: make(map[string]LightspeedInfo),
	}
}

type datastoreImpl struct {
	mu    sync.RWMutex
	store map[string]LightspeedInfo
}

func (d *datastoreImpl) Update(clusterID string, info LightspeedInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store[clusterID] = info
}

func (d *datastoreImpl) Get(clusterID string) (LightspeedInfo, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	info, ok := d.store[clusterID]
	return info, ok
}

func (d *datastoreImpl) Remove(clusterID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.store, clusterID)
}

func (d *datastoreImpl) GetAll() map[string]LightspeedInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]LightspeedInfo, len(d.store))
	for k, v := range d.store {
		result[k] = v
	}
	return result
}
