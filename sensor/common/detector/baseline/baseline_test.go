package baseline

import (
	"flag"
	"fmt"
	"runtime"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

var (
	benchMax = flag.Bool("bench.max", false, "Run maximum scale benchmarks (300k containers)")
)

// benchmarkMemoryUsage is a parameterized benchmark function
func benchmarkMemoryUsage(b *testing.B, evaluatorFactory func() Evaluator, baselines []*storage.ProcessBaseline, scenarioName string, showDeduplication bool) {
	runtime.GC()
	runtime.GC()
	
	var m1, m2 runtime.MemStats
	runtime.ReadMemStats(&m1)
	
	evaluator := evaluatorFactory()
	for _, baseline := range baselines {
		evaluator.AddBaseline(baseline)
	}
	
	runtime.GC()
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	memoryMB := float64(m2.HeapInuse-m1.HeapInuse) / (1024 * 1024)
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

func TestDeduplication(t *testing.T) {
	// Test that optimized implementation actually deduplicates
	optimized := newOptimizedBaselineEvaluator().(*optimizedBaselineEvaluator)

	// Create two identical baselines
	baseline1 := createTestBaseline("deployment-1", "container-1", 25)
	baseline2 := createTestBaseline("deployment-2", "container-2", 25)

	optimized.AddBaseline(baseline1)
	optimized.AddBaseline(baseline2)

	// Should have 2 deployment entries but only 1 process set
	assert.Equal(t, 2, len(optimized.deploymentBaselines))
	assert.Equal(t, 1, len(optimized.processSets))

	// Both deployments should reference the same process set
	key1 := optimized.deploymentBaselines["deployment-1"]["container-1"]
	key2 := optimized.deploymentBaselines["deployment-2"]["container-2"]
	assert.Equal(t, key1, key2)

	// Process set should have reference count of 2
	entry := optimized.processSets[key1] // key1 is now the content hash directly
	assert.Equal(t, 2, entry.refCount)
}

// BenchmarkBaselineEvaluator_Original_Identical tests original implementation with identical containers
func BenchmarkBaselineEvaluator_Original_Identical(b *testing.B) {
	containerCount := 10000
	scenarioName := "Original Identical_10k"
	if *benchMax {
		containerCount = 300000
		scenarioName = "Original Identical_300k"
	}
	baselines := createDuplicateBaselines(containerCount, 25)
	benchmarkMemoryUsage(b, newBaselineEvaluator, baselines, scenarioName, false)
}

// BenchmarkBaselineEvaluator_Optimized_Identical tests optimized implementation with identical containers
func BenchmarkBaselineEvaluator_Optimized_Identical(b *testing.B) {
	containerCount := 10000
	scenarioName := "Optimized Identical_10k"
	if *benchMax {
		containerCount = 300000
		scenarioName = "Optimized Identical_300k"
	}
	baselines := createDuplicateBaselines(containerCount, 25)
	benchmarkMemoryUsage(b, newOptimizedBaselineEvaluator, baselines, scenarioName, true)
}

// BenchmarkBaselineEvaluator_Original_Mixed tests original implementation with mixed containers
func BenchmarkBaselineEvaluator_Original_Mixed(b *testing.B) {
	containerCount := 10000
	imageTypes := 10
	scenarioName := "Original Mixed_10k"
	if *benchMax {
		containerCount = 300000
		imageTypes = 100
		scenarioName = "Original Mixed_300k"
	}
	baselines := createK8sRealisticBaselines(containerCount, 25, imageTypes)
	benchmarkMemoryUsage(b, newBaselineEvaluator, baselines, scenarioName, false)
}

// BenchmarkBaselineEvaluator_Optimized_Mixed tests optimized implementation with mixed containers
func BenchmarkBaselineEvaluator_Optimized_Mixed(b *testing.B) {
	containerCount := 10000
	imageTypes := 10
	scenarioName := "Optimized Mixed_10k"
	if *benchMax {
		containerCount = 300000
		imageTypes = 100
		scenarioName = "Optimized Mixed_300k"
	}
	baselines := createK8sRealisticBaselines(containerCount, 25, imageTypes)
	benchmarkMemoryUsage(b, newOptimizedBaselineEvaluator, baselines, scenarioName, true)
}

// BenchmarkBaselineEvaluator_Original_Unique tests original implementation with unique containers
func BenchmarkBaselineEvaluator_Original_Unique(b *testing.B) {
	containerCount := 10000
	scenarioName := "Original Unique_10k"
	if *benchMax {
		containerCount = 300000
		scenarioName = "Original Unique_300k"
	}
	baselines := createUniqueBaselines(containerCount, 25)
	benchmarkMemoryUsage(b, newBaselineEvaluator, baselines, scenarioName, false)
}

// BenchmarkBaselineEvaluator_Optimized_Unique tests optimized implementation with unique containers
func BenchmarkBaselineEvaluator_Optimized_Unique(b *testing.B) {
	containerCount := 10000
	scenarioName := "Optimized Unique_10k"
	if *benchMax {
		containerCount = 300000
		scenarioName = "Optimized Unique_300k"
	}
	baselines := createUniqueBaselines(containerCount, 25)
	benchmarkMemoryUsage(b, newOptimizedBaselineEvaluator, baselines, scenarioName, true)
}

func TestBaseline(t *testing.T) {
	testCases := []struct {
		name             string
		evaluatorFactory func() Evaluator
	}{
		{
			name: "Original",
			evaluatorFactory: func() Evaluator {
				return newBaselineEvaluator()
			},
		},
		{
			name: "Optimized",
			evaluatorFactory: func() Evaluator {
				return newOptimizedBaselineEvaluator()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			testBaselineImplementation(t, tc.evaluatorFactory)
		})
	}
}

func testBaselineImplementation(t *testing.T, evaluatorFactory func() Evaluator) {
	process := fixtures.GetProcessIndicator()

	notInUnlockedBaseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		},
	}

	notInBaseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		},
		UserLockedTimestamp: protocompat.TimestampNow(),
	}

	inBaseline := &storage.ProcessBaseline{
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		},
		Elements: []*storage.BaselineElement{
			{
				Element: &storage.BaselineItem{
					Item: &storage.BaselineItem_ProcessName{
						ProcessName: process.GetSignal().GetExecFilePath(),
					},
				},
			},
		},
		UserLockedTimestamp: protocompat.TimestampNow(),
	}

	evaluator := evaluatorFactory()
	// No baseline added, nothing is outside a locked baseline
	assert.False(t, evaluator.IsOutsideLockedBaseline(process))

	// Add baseline that does not contain the value
	evaluator.AddBaseline(notInBaseline)
	assert.True(t, evaluator.IsOutsideLockedBaseline(process))

	// Verify that different baselines produce expected outcomes.
	evaluator.AddBaseline(inBaseline)
	assert.False(t, evaluator.IsOutsideLockedBaseline(process))
	evaluator.AddBaseline(notInBaseline)
	assert.True(t, evaluator.IsOutsideLockedBaseline(process))
	evaluator.AddBaseline(notInUnlockedBaseline)
	assert.False(t, evaluator.IsOutsideLockedBaseline(process))

	// Add locked baseline then remove deployment
	evaluator.AddBaseline(notInBaseline)
	assert.True(t, evaluator.IsOutsideLockedBaseline(process))
	evaluator.RemoveDeployment(process.GetDeploymentId())
	assert.False(t, evaluator.IsOutsideLockedBaseline(process))
}

// createTestBaseline creates a process baseline with specified number of processes
func createTestBaseline(deploymentID, containerName string, processCount int) *storage.ProcessBaseline {
	elements := make([]*storage.BaselineElement, processCount)
	for i := 0; i < processCount; i++ {
		elements[i] = &storage.BaselineElement{
			Element: &storage.BaselineItem{
				Item: &storage.BaselineItem_ProcessName{
					ProcessName: fmt.Sprintf("/usr/bin/process-%d", i),
				},
			},
		}
	}

	return &storage.ProcessBaseline{
		Id: fmt.Sprintf("baseline-%s-%s", deploymentID, containerName),
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  deploymentID,
			ContainerName: containerName,
		},
		Elements:            elements,
		UserLockedTimestamp: protocompat.TimestampNow(),
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
