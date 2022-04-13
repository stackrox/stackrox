package gatherers

import (
	"runtime"

	"github.com/stackrox/stackrox/pkg/telemetry/data"
	"github.com/stackrox/stackrox/pkg/version"
)

// ComponentInfoGatherer gathers generic information about a StackRox component(Centra, Scanner, etc...)
type ComponentInfoGatherer struct {
}

// NewComponentInfoGatherer creates and returns a ComponentInfoGatherer
func NewComponentInfoGatherer() *ComponentInfoGatherer {
	return &ComponentInfoGatherer{}
}

// Gather returns generic telemetry information about a StackRox component (Central, Scanner, etc...)
func (c *ComponentInfoGatherer) Gather() *data.RoxComponentInfo {
	return &data.RoxComponentInfo{
		Version:  version.GetMainVersion(),
		Process:  getProcessInfo(),
		Restarts: 0, // TODO: Figure out how to get number of restarts
	}
}

func getProcessInfo() *data.ProcessInfo {
	return &data.ProcessInfo{
		NumGoroutines: runtime.NumGoroutine(),
		NumCPUs:       runtime.NumCPU(),
		Memory:        getMemInfo(),
	}
}

func getMemInfo() *data.ProcessMemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return &data.ProcessMemInfo{
		CurrentAllocBytes:   int64(m.Alloc),
		CurrentAllocObjects: int64(m.HeapObjects),
		TotalAllocBytes:     int64(m.TotalAlloc),
		TotalAllocObjects:   int64(m.Mallocs),
		SysMemBytes:         int64(m.Sys),
		NumGCs:              int64(m.NumGC),
		GCFraction:          m.GCCPUFraction,
	}
}
