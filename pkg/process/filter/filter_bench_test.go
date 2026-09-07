package filter

import (
	"fmt"
	"math"
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
	}
}

// BenchmarkHighCardinalityProcesses measures a single container accumulating many
// distinct executable paths. Sensor runs with an unbounded maxUniqueProcesses, so
// the process level must stay near O(1) per insert rather than degrading to O(n^2).
// The "maxed" case additionally pushes the configurable parameters to their limits
// (fanOut 255 across 10 levels, maxExact 1000) with several argument variations per
// process to also stress the argument-level tree.
func BenchmarkHighCardinalityProcesses(b *testing.B) {
	cases := []struct {
		name         string
		numProcesses int
		argsPerProc  int
		fanOut       []int
		maxExact     int
	}{
		{
			name:         "10k_processes_default_fanout",
			numProcesses: 10000,
			argsPerProc:  1,
			fanOut:       []int{8, 6, 4, 2},
			maxExact:     5,
		},
		{
			name:         "10k_processes_maxed_params",
			numProcesses: 10000,
			argsPerProc:  20,
			fanOut:       []int{255, 255, 255, 255, 255, 255, 255, 255, 255, 255},
			maxExact:     1000,
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			indicators := make([]*storage.ProcessIndicator, 0, tc.numProcesses*tc.argsPerProc)
			for p := range tc.numProcesses {
				for a := range tc.argsPerProc {
					indicators = append(indicators, &storage.ProcessIndicator{
						DeploymentId:  "dep",
						ContainerName: "container",
						Signal: &storage.ProcessSignal{
							ContainerId:  "id",
							ExecFilePath: fmt.Sprintf("/usr/bin/process%d", p),
							Args:         fmt.Sprintf("--flag%d value subcommand run", a),
						},
					})
				}
			}

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				filter := NewFilter(tc.maxExact, math.MaxInt, tc.fanOut)
				for _, pi := range indicators {
					filter.Add(pi)
				}
			}
		})
	}
}
