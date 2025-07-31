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

// memoryTracker collects memory measurements and reports aggregated statistics
type memoryTracker struct {
	measurements []float64
}

func (mt *memoryTracker) addMeasurement(memUsedMB float64) {
	mt.measurements = append(mt.measurements, memUsedMB)
}

func (mt *memoryTracker) reportStats(b *testing.B) {
	if len(mt.measurements) == 0 {
		return
	}
	
	var total, min, max float64
	min = mt.measurements[0]
	max = mt.measurements[0]
	
	for _, mem := range mt.measurements {
		total += mem
		if mem < min {
			min = mem
		}
		if mem > max {
			max = mem
		}
	}
	
	avg := total / float64(len(mt.measurements))
	b.Logf("Memory usage - Avg: %.1f MB, Min: %.1f MB, Max: %.1f MB (%d iterations)", avg, min, max, len(mt.measurements))
}

var (
	benchMax  = flag.Bool("bench.max", false, "Run maximum scale benchmarks (300k containers)")
	benchFull = flag.Bool("bench.full", false, "Run all benchmarks including maximum scale")
)

func TestBaseline(t *testing.T) {
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

	evaluator := NewBaselineEvaluator()
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
		// Copy elements for each baseline
		elementsCopy := make([]*storage.BaselineElement, len(elements))
		copy(elementsCopy, elements)
		
		baselines[i] = &storage.ProcessBaseline{
			Id: fmt.Sprintf("duplicate-baseline-%d", i),
			Key: &storage.ProcessBaselineKey{
				DeploymentId:  fmt.Sprintf("deployment-%d", i/10), // 10 containers per deployment
				ContainerName: fmt.Sprintf("container-%d", i%10),
			},
			Elements:            elementsCopy,
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

// BenchmarkBaselineEvaluator_AddBaseline benchmarks adding baselines
func BenchmarkBaselineEvaluator_AddBaseline(b *testing.B) {
	benchmarks := []struct {
		name         string
		baselineCount int
		processCount  int
	}{
		{"Small_10baselines_10processes", 10, 10},
		{"Medium_100baselines_50processes", 100, 50},
		{"Large_1000baselines_100processes", 1000, 100},
		{"XLarge_5000baselines_200processes", 5000, 200},
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			baselines := createDuplicateBaselines(bm.baselineCount, bm.processCount)
			
			b.ResetTimer()
			b.ReportAllocs()
			
			for i := 0; i < b.N; i++ {
				evaluator := NewBaselineEvaluator()
				for _, baseline := range baselines {
					evaluator.AddBaseline(baseline)
				}
			}
		})
	}
}

// BenchmarkBaselineEvaluator_MemoryUsage benchmarks memory usage targeting high memory allocations
func BenchmarkBaselineEvaluator_MemoryUsage(b *testing.B) {
	// CI-friendly quick benchmarks (always run)
	quickBenchmarks := []struct {
		name         string
		baselineCount int
		processCount  int
		description   string
	}{
		{"K8s_Small_10k_containers", 10000, 50, "Small K8s: 10k containers × 50 processes (~38MB allocated)"},
	}
	
	// Maximum scale benchmarks (requires -bench.max flag)
	maxBenchmarks := []struct {
		name         string
		baselineCount int
		processCount  int
		description   string
	}{
		{"K8s_Large_100k_containers", 100000, 50, "Large K8s: 100k containers × 50 processes (~381MB allocated)"},
		{"K8s_MaxScale_150k_containers", 150000, 50, "Max K8s scale: 150k containers × 50 processes (~572MB allocated)"},
		{"K8s_MaxScale_150k_unique", 150000, 50, "Max K8s scale: 150k containers ALL UNIQUE (~572MB allocated)"},
	}
	
	// Combine benchmarks based on flags
	var benchmarks []struct {
		name         string
		baselineCount int
		processCount  int
		description   string
	}
	
	benchmarks = append(benchmarks, quickBenchmarks...)
	
	if *benchMax || *benchFull {
		benchmarks = append(benchmarks, maxBenchmarks...)
	}
	
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			// Measure memory before
			var m1 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)
			
			var baselines []*storage.ProcessBaseline
			if bm.name == "K8s_MaxScale_150k_unique" {
				// Create fully unique baselines for this special case
				baselines = make([]*storage.ProcessBaseline, bm.baselineCount)
				for i := 0; i < bm.baselineCount; i++ {
					baselines[i] = createTestBaseline(
						fmt.Sprintf("deployment-%d", i),
						"container-0",
						bm.processCount,
					)
				}
			} else {
				// Use duplicate baselines for all other scenarios
				baselines = createDuplicateBaselines(bm.baselineCount, bm.processCount)
			}
			
			b.ResetTimer()
			b.ReportAllocs()
			
			tracker := &memoryTracker{}
			
			for i := 0; i < b.N; i++ {
				evaluator := NewBaselineEvaluator()
				
				// Add all baselines
				for _, baseline := range baselines {
					evaluator.AddBaseline(baseline)
				}
				
				// Measure memory after
				var m2 runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&m2)
				
				// Calculate memory correctly, handling potential underflow
				var memUsedMB float64
				if m2.Alloc > m1.Alloc {
					memUsedMB = float64(m2.Alloc-m1.Alloc) / (1024 * 1024)
				} else {
					memUsedMB = float64(m2.Alloc) / (1024 * 1024)
				}
				
				tracker.addMeasurement(memUsedMB)
				
				// Keep evaluator alive to measure peak memory
				_ = evaluator
			}
			
			tracker.reportStats(b)
		})
	}
}

// BenchmarkBaselineEvaluator_IsOutsideLockedBaseline benchmarks lookup performance
func BenchmarkBaselineEvaluator_IsOutsideLockedBaseline(b *testing.B) {
	// Create evaluator with many baselines
	evaluator := NewBaselineEvaluator()
	baselines := createDuplicateBaselines(1000, 100)
	
	for _, baseline := range baselines {
		evaluator.AddBaseline(baseline)
	}
	
	// Create test process indicator
	testProcess := &storage.ProcessIndicator{
		DeploymentId:  "deployment-0",
		ContainerName: "container-0",
		Signal: &storage.ProcessSignal{
			ExecFilePath: "/usr/bin/common-process-0",
		},
	}
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		_ = evaluator.IsOutsideLockedBaseline(testProcess)
	}
}

// BenchmarkBaselineEvaluator_DuplicationScenarios benchmarks different duplication patterns
func BenchmarkBaselineEvaluator_DuplicationScenarios(b *testing.B) {
	// Quick scenarios (always run in CI)
	quickScenarios := []struct {
		name        string
		setupFunc   func() []*storage.ProcessBaseline
		description string
	}{
		{
			name: "Duplicate_10k_containers",
			setupFunc: func() []*storage.ProcessBaseline {
				return createDuplicateBaselines(10000, 50)
			},
			description: "10k containers with identical process sets (~38MB allocated)",
		},
		{
			name: "Unique_10k_containers",
			setupFunc: func() []*storage.ProcessBaseline {
				baselines := make([]*storage.ProcessBaseline, 10000)
				for i := 0; i < 10000; i++ {
					baselines[i] = createTestBaseline(
						fmt.Sprintf("deployment-%d", i),
						"container-0",
						50,
					)
				}
				return baselines
			},
			description: "10k containers with unique process sets (~38MB allocated)",
		},
	}
	
	// Maximum scenarios (requires -bench.max flag)
	maxScenarios := []struct {
		name        string
		setupFunc   func() []*storage.ProcessBaseline
		description string
	}{
		{
			name: "Duplicate_300k_containers",
			setupFunc: func() []*storage.ProcessBaseline {
				return createDuplicateBaselines(300000, 50)
			},
			description: "Max K8s scale: 300k containers with identical process sets (~1.1GB allocated)",
		},
		{
			name: "Unique_300k_containers",
			setupFunc: func() []*storage.ProcessBaseline {
				baselines := make([]*storage.ProcessBaseline, 300000)
				for i := 0; i < 300000; i++ {
					baselines[i] = createTestBaseline(
						fmt.Sprintf("deployment-%d", i),
						"container-0",
						50,
					)
				}
				return baselines
			},
			description: "Max K8s scale: 300k containers with unique process sets (~1.1GB allocated)",
		},
		{
			name: "Microservices_300k_containers_10_images",
			setupFunc: func() []*storage.ProcessBaseline {
				return createK8sRealisticBaselines(300000, 50, 10)
			},
			description: "Microservices: 300k containers from 10 image types (30k copies each) (~1.1GB allocated)",
		},
		{
			name: "Realistic_300k_containers_200_images",
			setupFunc: func() []*storage.ProcessBaseline {
				return createK8sRealisticBaselines(300000, 50, 200)
			},
			description: "Realistic K8s: 300k containers from 200 image types (1.5k copies each) (~1.1GB allocated)",
		},
	}
	
	// Combine scenarios based on flags
	var scenarios []struct {
		name        string
		setupFunc   func() []*storage.ProcessBaseline
		description string
	}
	
	scenarios = append(scenarios, quickScenarios...)
	
	if *benchMax || *benchFull {
		scenarios = append(scenarios, maxScenarios...)
	}
	
	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			baselines := scenario.setupFunc()
			
			b.ResetTimer()
			b.ReportAllocs()
			
			tracker := &memoryTracker{}
			
			for i := 0; i < b.N; i++ {
				evaluator := NewBaselineEvaluator()
				
				var m1, m2 runtime.MemStats
				runtime.GC()
				runtime.ReadMemStats(&m1)
				
				for _, baseline := range baselines {
					evaluator.AddBaseline(baseline)
				}
				
				runtime.GC()
				runtime.ReadMemStats(&m2)
				
				// Calculate memory correctly, handling potential underflow
				var memUsedMB float64
				if m2.Alloc > m1.Alloc {
					memUsedMB = float64(m2.Alloc-m1.Alloc) / (1024 * 1024)
				} else {
					memUsedMB = float64(m2.Alloc) / (1024 * 1024)
				}
				
				tracker.addMeasurement(memUsedMB)
			}
			
			tracker.reportStats(b)
		})
	}
}
