package injector

import "github.com/stackrox/rox/generated/storage"

func average(metrics ...*storage.Usage) *storage.Usage {
	n := int32(len(metrics))
	a := &storage.Usage{}
	if n == 0 {
		return a
	}
	for _, m := range metrics {
		a.NumNodes += m.NumNodes
		a.NumCpuUnits += m.NumCpuUnits
	}
	a.NumNodes /= n
	a.NumCpuUnits /= n
	return a
}
