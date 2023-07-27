package datastore

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// cache holds in-memory cache of collected usage metrics.
type cache struct {
	// lastKnown stores the last known metrics per cluster.
	lastKnown    map[string]storage.Usage
	lastKnownMux sync.Mutex

	// nodesMap and coresMap store the maximum numbers of nodes and cores
	// per cluster. Maps access is internally synchronized.
	nodesMap maputil.CmpMap[string, int32]
	coresMap maputil.CmpMap[string, int32]
}

// NewCache initializes and returns a usage structure.
func NewCache() *cache {
	return &cache{
		lastKnown: make(map[string]storage.Usage),
		nodesMap:  maputil.NewMaxMap[string, int32](),
		coresMap:  maputil.NewMaxMap[string, int32](),
	}
}

func (u *cache) UpdateUsage(clusterID string, cm *central.ClusterMetrics) {
	u.nodesMap.Store(clusterID, int32(cm.GetNodeCount()))
	u.coresMap.Store(clusterID, int32(cm.GetCpuCapacity()))

	u.lastKnownMux.Lock()
	defer u.lastKnownMux.Unlock()
	u.lastKnown[clusterID] = storage.Usage{
		NumNodes: int32(cm.GetNodeCount()),
		NumCores: int32(cm.GetCpuCapacity()),
	}
}

// FilterCurrent removes the last known metrics values for the cluster IDs
// not present in the ids, and returns the total values for other IDs.
func (u *cache) FilterCurrent(ids set.StringSet) *storage.Usage {
	m := storage.Usage{}
	u.lastKnownMux.Lock()
	defer u.lastKnownMux.Unlock()
	for id, v := range u.lastKnown {
		if ids.Contains(id) {
			m.NumNodes += v.NumNodes
			m.NumCores += v.NumCores
		} else {
			delete(u.lastKnown, id)
		}
	}
	return &m
}

// CutMetrics resets the metrics and returns the collected values since last
// invocation.
// The cluster ids are provided to filter the result, so it aggregates only the
// collected metrics of the currently known clusters. This is to avoid double
// usage counting when customers remove and add clusters within one collection
// period.
func (u *cache) CutMetrics(ids set.StringSet) *storage.Usage {
	m := storage.Usage{}
	for id, v := range u.nodesMap.Reset() {
		if ids.Contains(id) {
			m.NumNodes += v
		}
	}
	for id, v := range u.coresMap.Reset() {
		if ids.Contains(id) {
			m.NumCores += v
		}
	}
	return &m
}
