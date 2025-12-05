package filter

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

// BenchmarkAdd measures performance of adding process indicators
func BenchmarkAdd(b *testing.B) {
	filter := NewFilter(100, 1000, []int{100, 100, 100})

	// Create test data
	indicators := make([]*storage.ProcessIndicator, 1000)
	for i := range indicators {
		indicators[i] = &storage.ProcessIndicator{
			DeploymentId:  fmt.Sprintf("dep%d", i%10),
			ContainerName: "container",
			Signal: &storage.ProcessSignal{
				ContainerId:  fmt.Sprintf("id%d", i%10),
				ExecFilePath: fmt.Sprintf("/usr/bin/process%d", i%100),
				Args:         fmt.Sprintf("arg1 arg2 arg3 iteration%d", i),
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Add(indicators[i%len(indicators)])
	}
}

// BenchmarkAddMemory measures memory allocations
func BenchmarkAddMemory(b *testing.B) {
	filter := NewFilter(100, 1000, []int{100, 100, 100})

	pi := &storage.ProcessIndicator{
		DeploymentId:  "deployment",
		ContainerName: "container",
		Signal: &storage.ProcessSignal{
			ContainerId:  "containerid",
			ExecFilePath: "/usr/bin/process",
			Args:         "arg1 arg2 arg3",
		},
	}

	for b.Loop() {
		filter.Add(pi)
	}
}

// BenchmarkBuildIndicatorFilterMemory measures memory usage when building a filter
// with a large number of processes
func BenchmarkBuildIndicatorFilterMemory(b *testing.B) {
	const (
		NumDeployments       = 100
		NumPodsPerDeployment = 10
		NumProcessesPerPod   = 10
	)

	for b.Loop() {
		filter := NewFilter(1000, 10000, []int{100, 50, 25, 10, 5})

		for i := 0; i < NumDeployments; i++ {
			deploymentID := fmt.Sprintf("deployment-%d", i)
			for j := 0; j < NumPodsPerDeployment; j++ {
				containerID := fmt.Sprintf("container-%d-%d", i, j)
				for k := 0; k < NumProcessesPerPod; k++ {
					pi := &storage.ProcessIndicator{
						DeploymentId:  deploymentID,
						ContainerName: "container",
						Signal: &storage.ProcessSignal{
							ContainerId:  containerID,
							ExecFilePath: fmt.Sprintf("/usr/bin/process%d", k),
							Args:         fmt.Sprintf("arg1 arg2 arg3 iteration%d", k),
						},
					}
					filter.Add(pi)
				}
			}
		}

		// Force GC to measure actual memory retained
		runtime.GC()
	}
}
