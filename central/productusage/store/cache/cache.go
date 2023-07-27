package cache

import (
	gogoTypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
)

// cacheImpl holds in-memory cache of collected usage metrics.
type cacheImpl struct {
	// lastKnown stores the last known metrics per cluster.
	lastKnown maputil.SyncMap[string, *storage.SecuredUnits]
	// nodesMap stores the maximum numbers of nodes per cluster.
	nodesMap maputil.SyncMap[string, int64]
	// cpuUnitsMap stores the maximum numbers of CPU Units per cluster.
	cpuUnitsMap maputil.SyncMap[string, int64]
}

// Cache interface provides methods to manipulate a usage metrics cash.
type Cache interface {
	// UpdateUsage upserts the metrics to the cache for the given cluster id,
	// keeping maximum and last values.
	UpdateUsage(id string, cm *storage.SecuredUnits)
	// Cleanup removes the records of the clusters, which are not in the ids set.
	Cleanup(ids set.StringSet)
	// GetCurrent returns the collected values.
	GetCurrent() *storage.SecuredUnits
	// AggregateAndFlush returns the maximum usage values and resets them in the cache.
	AggregateAndFlush() *storage.SecuredUnits
}

// NewCache initializes and returns a cache instance.
func NewCache() Cache {
	return &cacheImpl{
		lastKnown:   maputil.NewSyncMap[string, *storage.SecuredUnits](),
		nodesMap:    maputil.NewMaxMap[string, int64](),
		cpuUnitsMap: maputil.NewMaxMap[string, int64](),
	}
}

func (u *cacheImpl) UpdateUsage(id string, cm *storage.SecuredUnits) {
	u.nodesMap.Store(id, cm.GetNumNodes())
	u.cpuUnitsMap.Store(id, cm.GetNumCpuUnits())
	u.lastKnown.Store(id, cm)
}

// getFilter returns a filter function, that removes all keys absent in the
// provided set.
func getFilter[T any](ids set.StringSet) func(m *map[string]T) {
	return func(mptr *map[string]T) {
		m := *mptr
		for key := range m {
			if !ids.Contains(key) {
				delete(m, key)
			}
		}
	}
}

// Cleanup removes the last known metrics values for the cluster IDs
// not present in the ids, and returns the total values for other IDs.
func (u *cacheImpl) Cleanup(ids set.StringSet) {
	{
		fn := getFilter[int64](ids)
		u.nodesMap.Access(fn)
		u.cpuUnitsMap.Access(fn)
	}
	fn := getFilter[*storage.SecuredUnits](ids)
	u.lastKnown.Access(fn)
}

// GetCurrent returns the total of the collected values.
func (u *cacheImpl) GetCurrent() *storage.SecuredUnits {
	var result storage.SecuredUnits
	u.lastKnown.RAccess(func(m map[string]*storage.SecuredUnits) {
		for _, v := range m {
			result.NumNodes += v.NumNodes
			result.NumCpuUnits += v.NumCpuUnits
		}
	})
	return &result
}

// AggregateAndFlush resets the metrics and returns the collected values since
// the previous invocation.
// The cluster ids are provided to Cleanup the result, so it aggregates only the
// collected metrics of the currently known clusters. This is to avoid double
// usage counting when customers remove and add clusters within one collection
// period.
func (u *cacheImpl) AggregateAndFlush() *storage.SecuredUnits {
	result := storage.SecuredUnits{
		Timestamp: gogoTypes.TimestampNow(),
	}
	u.nodesMap.Access(func(m *map[string]int64) {
		for _, v := range *m {
			result.NumNodes += v
		}
		*m = make(map[string]int64)
	})
	u.cpuUnitsMap.Access(func(m *map[string]int64) {
		for _, v := range *m {
			result.NumCpuUnits += v
		}
		*m = make(map[string]int64)
	})
	return &result
}
