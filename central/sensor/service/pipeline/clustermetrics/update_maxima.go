package clustermetrics

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/set"
)

// BillingMetrics are the metrics we collect and show to the customers to help
// them report their usage.
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
func CutMetrics(ids set.StringSet) *BillingMetrics {
	var m BillingMetrics
	for id, v := range nodesMap.Reset() {
		if ids.Contains(id) {
			m.TotalNodes += v
		}
	}
	for id, v := range millicoresMap.Reset() {
		if ids.Contains(id) {
			m.TotalMilliCores += v
		}
	}
	return &m
}
