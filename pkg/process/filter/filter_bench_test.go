package filter

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/stackrox/rox/generated/storage"
)

func BenchmarkAdd(b *testing.B) {
	cases := []struct {
		name           string
		fanOut         []int
		maxExact       int
		maxUnique      int
		numIndicators  int
		numDeployments int
		numProcesses   int
	}{
		{
			name:           "default_fanout",
			fanOut:         []int{8, 6, 4, 2},
			maxExact:       5,
			maxUnique:      5000,
			numIndicators:  1000,
			numDeployments: 10,
			numProcesses:   100,
		},
		{
			name:           "high_fanout",
			fanOut:         []int{100, 100, 100},
			maxExact:       100,
			maxUnique:      1000,
			numIndicators:  1000,
			numDeployments: 10,
			numProcesses:   100,
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			filter := NewFilter(tc.maxExact, tc.maxUnique, tc.fanOut)

			indicators := make([]*storage.ProcessIndicator, tc.numIndicators)
			for i := range indicators {
				indicators[i] = &storage.ProcessIndicator{
					DeploymentId:  fmt.Sprintf("dep%d", i%tc.numDeployments),
					ContainerName: "container",
					Signal: &storage.ProcessSignal{
						ContainerId:  fmt.Sprintf("id%d", i%tc.numDeployments),
						ExecFilePath: fmt.Sprintf("/usr/bin/process%d", i%tc.numProcesses),
						Args:         fmt.Sprintf("arg1 arg2 arg3 iteration%d", i),
					},
				}
			}

			for i := 0; b.Loop(); i++ {
				filter.Add(indicators[i%len(indicators)])
			}
		})
	}
}

func BenchmarkLookupInFullLevel(b *testing.B) {
	cases := []struct {
		name    string
		fanOut  []int
		numArgs int
	}{
		{name: "fanout_8", fanOut: []int{8}, numArgs: 8},
		{name: "fanout_20", fanOut: []int{20}, numArgs: 20},
		{name: "fanout_100", fanOut: []int{100}, numArgs: 100},
		{name: "fanout_255", fanOut: []int{255}, numArgs: 255},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			filter := NewFilter(tc.numArgs+1, 5000, tc.fanOut)

			for i := range tc.numArgs {
				filter.Add(&storage.ProcessIndicator{
					DeploymentId:  "dep",
					ContainerName: "container",
					Signal: &storage.ProcessSignal{
						ContainerId:  "id",
						ExecFilePath: "/usr/bin/proc",
						Args:         fmt.Sprintf("arg%d", i),
					},
				})
			}

			last := &storage.ProcessIndicator{
				DeploymentId:  "dep",
				ContainerName: "container",
				Signal: &storage.ProcessSignal{
					ContainerId:  "id",
					ExecFilePath: "/usr/bin/proc",
					Args:         fmt.Sprintf("arg%d", tc.numArgs-1),
				},
			}

			for b.Loop() {
				filter.Add(last)
			}
		})
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

		for i := range NumDeployments {
			deploymentID := fmt.Sprintf("deployment-%d", i)
			for j := range NumPodsPerDeployment {
				containerID := fmt.Sprintf("container-%d-%d", i, j)
				for k := range NumProcessesPerPod {
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
