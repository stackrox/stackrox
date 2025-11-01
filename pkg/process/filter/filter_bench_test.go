package filter

import (
	"runtime"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
)

// BenchmarkFilter tests typical usage patterns: diverse workload with mixed operations
func BenchmarkFilter(b *testing.B) {
	filter := NewFilter(100, 100, []int{50, 50, 50})

	// Pre-create diverse indicators simulating real-world usage
	piCount := 100000
	indicators := make([]*storage.ProcessIndicator, piCount)
	for i := 0; i < piCount; i++ {
		pi := fixtures.GetProcessIndicator()
		pi.DeploymentId = pi.GetDeploymentId() + string(rune(i%100))
		pi.Signal.ContainerId = "container" + string(rune(i%1000))
		pi.Signal.ExecFilePath = "/bin/proc" + string(rune(i%500))
		pi.Signal.Args = "arg1 arg2 arg" + string(rune(i%2000))
		indicators[i] = pi
	}

	// Track memory usage
	done := make(chan struct{})
	var maxHeap, maxHeapObjects uint64
	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		var m runtime.MemStats
		for {
			select {
			case <-ticker.C:
				runtime.GC()
				runtime.ReadMemStats(&m)
				if m.HeapAlloc > maxHeap {
					maxHeap = m.HeapAlloc
				}
				if m.HeapObjects > maxHeapObjects {
					maxHeapObjects = m.HeapObjects
				}
			case <-done:
				return
			}
		}
	}()

	var accepted, rejected int
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 90% adds, 10% deletes (realistic ratio)
		if i%10 == 0 {
			filter.Delete(indicators[i%1000].GetDeploymentId())
		} else {
			if filter.Add(indicators[i%1000]) {
				accepted++
			} else {
				rejected++
			}
		}
	}

	b.StopTimer()
	close(done)
	time.Sleep(20 * time.Millisecond) // Let tracker finish

	b.ReportMetric(float64(maxHeap)/(1024*1024), "max_heap_MiB")
	b.ReportMetric(float64(maxHeapObjects), "max_heap_objects")
	if accepted+rejected > 0 {
		b.ReportMetric(float64(accepted)/float64(accepted+rejected)*100, "accept_rate_%")
	}
}
