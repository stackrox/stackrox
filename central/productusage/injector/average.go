package injector

import "github.com/stackrox/rox/generated/storage"

func average(metrics ...*storage.SecuredUnits) *storage.SecuredUnits {
	n := int64(len(metrics))
	averageUsage := &storage.SecuredUnits{}
	if n == 0 {
		return averageUsage
	}
	for _, m := range metrics {
		averageUsage.NumNodes += m.NumNodes
		averageUsage.NumCpuUnits += m.NumCpuUnits
	}
	averageUsage.NumNodes /= n
	averageUsage.NumCpuUnits /= n
	return averageUsage
}
