package injector

import (
	datastore "github.com/stackrox/rox/central/productusage/datastore/securedunits"
)

func average(metrics ...datastore.Data) datastore.Data {
	n := int64(len(metrics))
	averageUsage := &datastore.DataImpl{}
	if n == 0 {
		return averageUsage
	}
	for _, m := range metrics {
		averageUsage.NumNodes += m.GetNumNodes()
		averageUsage.NumCpuUnits += m.GetNumCPUUnits()
	}
	averageUsage.NumNodes /= n
	averageUsage.NumCpuUnits /= n
	if len(metrics) != 0 {
		averageUsage.Timestamp = metrics[len(metrics)-1].GetTimestamp()
	}
	return averageUsage
}
