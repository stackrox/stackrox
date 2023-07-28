package cache

import (
	"github.com/stackrox/rox/central/usage/source"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

// cacheImpl holds in-memory cache of collected usage metrics.
type cacheImpl struct {
	// lastKnown stores the last known metrics per cluster.
	lastKnown    map[string]storage.Usage
	lastKnownMux sync.Mutex

	// nodesMap and cpuUnitsMap store the maximum numbers of nodes and cores
	// per cluster. Maps access is internally synchronized.
	nodesMap    maputil.CmpMap[string, int32]
	cpuUnitsMap maputil.CmpMap[string, int32]
}

// Cache interface provides methods to manipulate a usage metrics cash.
type Cache interface {
	// UpdateUsage upserts the metrics to the cache for the given cluster id,
	// keeping maximum and last values.
	UpdateUsage(id string, cm source.MetricsSource)
	// FilterCurrent filters the last known usage metrics by keeping only the
	// values of the clusters with the provided ids, and returns the values.
	FilterCurrent(ids set.StringSet) *storage.Usage
	// CutMetrics returns the maximum usage values and resets them in the cache.
	CutMetrics(ids set.StringSet) *storage.Usage
}

// NewCache initializes and returns a cache instance.
func NewCache() Cache {
	return &cacheImpl{
		lastKnown:   make(map[string]storage.Usage),
		nodesMap:    maputil.NewMaxMap[string, int32](),
		cpuUnitsMap: maputil.NewMaxMap[string, int32](),
	}
}

func (u *cacheImpl) UpdateUsage(id string, cm source.MetricsSource) {
	u.nodesMap.Store(id, int32(cm.GetNodeCount()))
	u.cpuUnitsMap.Store(id, int32(cm.GetCpuCapacity()))

	u.lastKnownMux.Lock()
	defer u.lastKnownMux.Unlock()
	u.lastKnown[id] = storage.Usage{
		NumNodes:    int32(cm.GetNodeCount()),
		NumCpuUnits: int32(cm.GetCpuCapacity()),
	}
}

// FilterCurrent removes the last known metrics values for the cluster IDs
// not present in the ids, and returns the total values for other IDs.
func (u *cacheImpl) FilterCurrent(ids set.StringSet) *storage.Usage {
	m := storage.Usage{}
	u.lastKnownMux.Lock()
	defer u.lastKnownMux.Unlock()
	for id, v := range u.lastKnown {
		if ids.Contains(id) {
			m.NumNodes += v.NumNodes
			m.NumCpuUnits += v.NumCpuUnits
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
func (u *cacheImpl) CutMetrics(ids set.StringSet) *storage.Usage {
	m := storage.Usage{}
	for id, v := range u.nodesMap.Reset() {
		if ids.Contains(id) {
			m.NumNodes += v
		}
	}
	for id, v := range u.cpuUnitsMap.Reset() {
		if ids.Contains(id) {
			m.NumCpuUnits += v
		}
	}
	return &m
}
