package telemetry

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/sync"
)

type mapType = map[string]*central.ClusterMetrics

func newMetricsCache() *metricsCache {
	return &metricsCache{
		cache: make(mapType),
		sum:   &central.ClusterMetrics{},
	}
}

type metricsCache struct {
	cache mapType
	mutex sync.RWMutex
	sum   *central.ClusterMetrics
}

func (c *metricsCache) Set(clusterID string, clusterMetrics *central.ClusterMetrics) {
	if c == nil {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	oldValue := c.cache[clusterID]
	c.cache[clusterID] = clusterMetrics
	c.sum.SetCpuCapacity(c.sum.GetCpuCapacity() + (clusterMetrics.GetCpuCapacity() - oldValue.GetCpuCapacity()))
	c.sum.SetNodeCount(c.sum.GetNodeCount() + (clusterMetrics.GetNodeCount() - oldValue.GetNodeCount()))
}

func (c *metricsCache) Delete(clusterID string) {
	if c == nil {
		return
	}
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if clusterMetric, ok := c.cache[clusterID]; ok {
		c.sum.SetCpuCapacity(c.sum.GetCpuCapacity() - clusterMetric.GetCpuCapacity())
		c.sum.SetNodeCount(c.sum.GetNodeCount() - clusterMetric.GetNodeCount())
		delete(c.cache, clusterID)
	}
}

func (c *metricsCache) Sum() *central.ClusterMetrics {
	if c == nil {
		return &central.ClusterMetrics{}
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	cm := &central.ClusterMetrics{}
	cm.SetCpuCapacity(c.sum.GetCpuCapacity())
	cm.SetNodeCount(c.sum.GetNodeCount())
	return cm
}

func (c *metricsCache) Len() int {
	if c == nil {
		return 0
	}
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.cache)
}
