package datastore

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
)

// DataStore stores per-cluster OpenShift Lightspeed availability state in memory.
type DataStore interface {
	Update(clusterID string, info *central.LightspeedInfo)
	Get(clusterID string) *central.LightspeedInfo
	Remove(clusterID string)
	GetAll() map[string]*central.LightspeedInfo
}

// New returns a new in-memory DataStore.
func New() DataStore {
	return &datastoreImpl{
		store: make(map[string]*central.LightspeedInfo),
	}
}

type datastoreImpl struct {
	mu    sync.RWMutex
	store map[string]*central.LightspeedInfo
}

func (d *datastoreImpl) Update(clusterID string, info *central.LightspeedInfo) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.store[clusterID] = info
}

func (d *datastoreImpl) Get(clusterID string) *central.LightspeedInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.store[clusterID]
}

func (d *datastoreImpl) Remove(clusterID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.store, clusterID)
}

func (d *datastoreImpl) GetAll() map[string]*central.LightspeedInfo {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make(map[string]*central.LightspeedInfo, len(d.store))
	for k, v := range d.store {
		result[k] = v
	}
	return result
}
