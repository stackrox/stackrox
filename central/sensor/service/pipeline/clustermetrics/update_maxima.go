package clustermetrics

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/maputil"
)

type BillingMetrics struct {
	TotalNodes      int64
	TotalMilliCores int64
}

var (
	nodesMap      = maputil.NewMaxMap[string, int64]()
	millicoresMap = maputil.NewMaxMap[string, int64]()
)

func updateMaxima(clusterID string, cm *central.ClusterMetrics) {
	nodesMap.Add(clusterID, cm.GetNodeCount())
	millicoresMap.Add(clusterID, cm.GetCpuCapacity())
}

// CutMetrics resets the metrics and returns the collected values since last
// invocation.
func CutMetrics() BillingMetrics {
	var newMetrics BillingMetrics
	for _, v := range nodesMap.Reset() {
		newMetrics.TotalNodes += v
	}
	for _, v := range millicoresMap.Reset() {
		newMetrics.TotalMilliCores += v
	}
	return newMetrics
}
