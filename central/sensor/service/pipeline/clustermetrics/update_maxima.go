package clustermetrics

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
)

// BillingMetrics are the metrics we collect and show to the customers to help
// them report their usage.
type BillingMetrics struct {
	TotalNodes int64
	TotalCores int64
}

var (
	// lastKnown stores the last known metrics per cluster.
	lastKnown = map[string]BillingMetrics{}

	// nodesMap and coresMap store the maximum numbers of nodes and cores
	// per cluster.
	nodesMap = maputil.NewMaxMap[string, int64]()
	coresMap = maputil.NewMaxMap[string, int64]()
)

func updateMaxima(clusterID string, cm *central.ClusterMetrics) {
	lastKnown[clusterID] = BillingMetrics{
		TotalNodes: cm.GetNodeCount(),
		TotalCores: cm.GetCpuCapacity(),
	}

	nodesMap.Add(clusterID, cm.GetNodeCount())
	coresMap.Add(clusterID, cm.GetCpuCapacity())
}

// FilterCurrent removes the last known metrics values for the cluster IDs
// not present in the ids, and returns the total values for other IDs.
func FilterCurrent(ids set.StringSet) *BillingMetrics {
	var m BillingMetrics
	for id, v := range lastKnown {
		if ids.Contains(id) {
			m.TotalNodes += v.TotalNodes
			m.TotalCores += v.TotalCores
		} else {
			delete(lastKnown, id)
		}
	}
	return &m
}

// CutMetrics resets the metrics and returns the collected values since last
// invocation.
func CutMetrics(ids set.StringSet) *BillingMetrics {
	var m BillingMetrics
	for id, v := range nodesMap.Reset() {
		if ids.Contains(id) {
			m.TotalNodes += v
		}
	}
	for id, v := range coresMap.Reset() {
		if ids.Contains(id) {
			m.TotalCores += v
		}
	}
	return &m
}
