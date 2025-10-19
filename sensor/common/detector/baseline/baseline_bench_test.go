package baseline

import (
	"flag"
	"fmt"
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
)

var (
	benchMax = flag.Bool("bench.max", false, "Run maximum scale benchmarks (300k containers)")
)

// benchmarkMemoryUsage is a parameterized benchmark function
func benchmarkMemoryUsage(b *testing.B, evaluatorFactory func() Evaluator, baselines []*storage.ProcessBaseline, scenarioName string, showDeduplication bool) {
	oldGCPercent := debug.SetGCPercent(-1)
	defer debug.SetGCPercent(oldGCPercent)

	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)

	evaluator := evaluatorFactory()
	for _, baseline := range baselines {
		evaluator.AddBaseline(baseline)
	}

	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)

	memoryMB := float64(m2.Alloc-m1.Alloc) / (1024 * 1024)
	//memoryMB := float64(m2.TotalAlloc-m1.TotalAlloc) / (1024 * 1024)
	b.Logf("%s Memory: %.1f MB", scenarioName, memoryMB)

	// Show deduplication stats for optimized implementation
	if showDeduplication {
		if opt, ok := evaluator.(*optimizedBaselineEvaluator); ok {
			var totalMappings, totalSharedSets int
			for _, containerMap := range opt.deploymentBaselines {
				totalMappings += len(containerMap)
			}
			totalSharedSets = len(opt.processSets)
			b.Logf("Deduplication: %d containers → %d shared sets", totalMappings, totalSharedSets)
		}
	}

	runtime.KeepAlive(evaluator)
	runtime.KeepAlive(baselines)
}

// benchmarkAddBaselinePerformance measures CPU performance of adding baselines
func benchmarkAddBaselinePerformance(b *testing.B, evaluatorFactory func() Evaluator, baselines []*storage.ProcessBaseline, scenarioName string) {
	for i := 0; i < b.N; i++ {
		evaluator := evaluatorFactory()
		for _, baseline := range baselines {
			evaluator.AddBaseline(baseline)
		}
	}
}

func BenchmarkAllMemory(b *testing.B) {
	imageTypes := 10
	containerCount := 10000
	if *benchMax {
		containerCount = 300000
	}
	duplicateBaselines := createDuplicateBaselines(containerCount, 25)
	k8sRealisticBaselines := createK8sRealisticBaselines(containerCount, 25, imageTypes)
	uniqueBaselines := createUniqueBaselines(containerCount, 25)

	testMap := make(map[string][]*storage.ProcessBaseline)

	testMap["Identical"] = duplicateBaselines
	testMap["Mixed"] = k8sRealisticBaselines
	testMap["Unique"] = uniqueBaselines

	evaluatorTypes := []string{"Original", "Optimized", "OptimizedRobby", "OptimizedXXHash", "OptimizedNoIntermediateStrings", "OptimizedNoIntermediateStringsXXHash", "OptimizedPtr"}

	for baselineType, baselines := range testMap {
		for _, evaluatorType := range evaluatorTypes {
			evaluatorFactory := func() Evaluator { return NewBaselineEvaluator(evaluatorType) }
			scenarioName := evaluatorType + " " + baselineType + "_10k"
			if *benchMax {
				scenarioName = evaluatorType + " " + baselineType + "_300k"
			}
			b.Run(scenarioName, func(b *testing.B) {
				benchmarkMemoryUsage(b, evaluatorFactory, baselines, scenarioName, true)
			})
		}
	}
}

// createDuplicateBaselines creates many baselines with identical process sets
func createDuplicateBaselines(baselineCount, processCount int) []*storage.ProcessBaseline {
	baselines := make([]*storage.ProcessBaseline, baselineCount)

	// Create identical process elements that will be duplicated
	elements := make([]*storage.BaselineElement, processCount)
	for i := 0; i < processCount; i++ {
		elements[i] = &storage.BaselineElement{
			Element: &storage.BaselineItem{
				Item: &storage.BaselineItem_ProcessName{
					ProcessName: fmt.Sprintf("/usr/bin/common-process-%d", i),
				},
			},
		}
	}

	// Create many baselines with the same process set
	for i := 0; i < baselineCount; i++ {
		baselines[i] = &storage.ProcessBaseline{
			Id: fmt.Sprintf("duplicate-baseline-%d", i),
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  fmt.Sprintf("deployment-%d", i/10), // 10 containers per deployment
				ContainerName: fmt.Sprintf("container-%d", i%10),
			},
			Elements:            elements,
			UserLockedTimestamp: protocompat.TimestampNow(),
		}
	}

	return baselines
}

// createK8sRealisticBaselines creates baselines simulating real K8s cluster with common container images
func createK8sRealisticBaselines(totalContainers int, processesPerContainer int, uniqueImageTypes int) []*storage.ProcessBaseline {
	baselines := make([]*storage.ProcessBaseline, totalContainers)
	containersPerImageType := totalContainers / uniqueImageTypes

	for i := 0; i < totalContainers; i++ {
		// Determine which "image type" this container represents
		imageTypeID := i / containersPerImageType
		if imageTypeID >= uniqueImageTypes {
			imageTypeID = uniqueImageTypes - 1 // Handle remainder containers
		}

		// Create baseline elements based on image type
		elements := make([]*storage.BaselineElement, processesPerContainer)
		for j := 0; j < processesPerContainer; j++ {
			elements[j] = &storage.BaselineElement{
				Element: &storage.BaselineItem{
					Item: &storage.BaselineItem_ProcessName{
						ProcessName: fmt.Sprintf("/usr/bin/imagetype-%d-process-%d", imageTypeID, j),
					},
				},
			}
		}

		baselines[i] = &storage.ProcessBaseline{
			Id: fmt.Sprintf("k8s-container-%d", i),
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  fmt.Sprintf("deployment-%d", i/10), // 10 containers per deployment
				ContainerName: fmt.Sprintf("container-%d", i%10),
			},
			Elements:            elements,
			UserLockedTimestamp: protocompat.TimestampNow(),
		}
	}

	return baselines
}

// createUniqueBaselines creates baselines with completely unique process sets
func createUniqueBaselines(baselineCount, processCount int) []*storage.ProcessBaseline {
	baselines := make([]*storage.ProcessBaseline, baselineCount)

	for i := 0; i < baselineCount; i++ {
		deploymentID := fmt.Sprintf("deployment-%d", i/10) // 10 containers per deployment
		containerName := fmt.Sprintf("container-%d", i%10)

		// Create completely unique process names that include deployment and container info
		elements := make([]*storage.BaselineElement, processCount)
		for j := 0; j < processCount; j++ {
			elements[j] = &storage.BaselineElement{
				Element: &storage.BaselineItem{
					Item: &storage.BaselineItem_ProcessName{
						ProcessName: fmt.Sprintf("/unique/%s/%s/process-%d", deploymentID, containerName, j),
					},
				},
			}
		}

		baselines[i] = &storage.ProcessBaseline{
			Id: fmt.Sprintf("unique-baseline-%d", i),
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  deploymentID,
				ContainerName: containerName,
			},
			Elements:            elements,
			UserLockedTimestamp: protocompat.TimestampNow(),
		}
	}

	return baselines
}

// CPU Performance Benchmarks - Measure computational overhead of adding baselines

func BenchmarkAddBaseline(b *testing.B) {
	containerCount := 1000
	imageTypes := 10
	duplicateBaselines := createDuplicateBaselines(containerCount, 25)
	k8sRealisticBaselines := createK8sRealisticBaselines(containerCount, 25, imageTypes)
	uniqueBaselines := createUniqueBaselines(containerCount, 25)

	testMap := make(map[string][]*storage.ProcessBaseline)

	testMap["Identical"] = duplicateBaselines
	testMap["Mixed"] = k8sRealisticBaselines
	testMap["Unique"] = uniqueBaselines

	evaluatorTypes := []string{"Original", "Optimized", "OptimizedRobby", "OptimizedXXHash", "OptimizedNoIntermediateStrings", "OptimizedNoIntermediateStringsXXHash", "OptimizedPtr"}

	for baselineType, baselines := range testMap {
		for _, evaluatorType := range evaluatorTypes {
			evaluatorFactory := func() Evaluator { return NewBaselineEvaluator(evaluatorType) }
			scenarioName := evaluatorType + " " + baselineType
			b.Run(scenarioName, func(b *testing.B) {
				benchmarkAddBaselinePerformance(b, evaluatorFactory, baselines, "Optimized Identical")
			})
		}
	}
}
