package baseline

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
)

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

func TestDeduplicationModifyBaseline(t *testing.T) {
	// Test that optimized implementation actually deduplicates
	optimized := newOptimizedBaselineEvaluator().(*optimizedBaselineEvaluator)

	// Create two identical baselines
	baseline1 := createTestBaseline("deployment-1", "container-1", 25)
	baseline2 := createTestBaseline("deployment-2", "container-2", 25)
	// Add a baseline for an existing container with one more process
	baseline3 := createTestBaseline("deployment-1", "container-1", 26)

	optimized.AddBaseline(baseline1)
	optimized.AddBaseline(baseline2)
	optimized.AddBaseline(baseline3)

	// Should have 2 deployment entries and two baselines
	assert.Equal(t, 2, len(optimized.deploymentBaselines))
	assert.Equal(t, 2, len(optimized.processSets))

	// Both deployments should reference different process sets
	key1 := optimized.deploymentBaselines["deployment-1"]["container-1"]
	key2 := optimized.deploymentBaselines["deployment-2"]["container-2"]
	assert.NotEqual(t, key1, key2)

	// Both process sets should have a ref count of 1
	entry1 := optimized.processSets[key1]
	assert.Equal(t, 1, entry1.refCount)
	entry2 := optimized.processSets[key2]
	assert.Equal(t, 1, entry2.refCount)

	// The second container also gets the same process so they should
	// share the same process set again
	baseline4 := createTestBaseline("deployment-2", "container-2", 26)
	optimized.AddBaseline(baseline4)

	// Should have 2 deployment entries but only 1 process set
	assert.Equal(t, 2, len(optimized.deploymentBaselines))
	assert.Equal(t, 1, len(optimized.processSets))

	// Both deployments should reference the same process set
	key1 = optimized.deploymentBaselines["deployment-1"]["container-1"]
	key2 = optimized.deploymentBaselines["deployment-2"]["container-2"]
	assert.Equal(t, key1, key2)

	// Process set should have reference count of 2
	entry := optimized.processSets[key1] // key1 is now the content hash directly
	assert.Equal(t, 2, entry.refCount)
}

func TestDeduplicationRemoveDeployment(t *testing.T) {
	// Test that optimized implementation actually deduplicates
	optimized := newOptimizedBaselineEvaluator().(*optimizedBaselineEvaluator)

	// Create two identical baselines
	baseline1 := createTestBaseline("deployment-1", "container-1", 25)
	baseline2 := createTestBaseline("deployment-2", "container-2", 25)
	baseline3 := createTestBaseline("deployment-1", "container-2", 25)

	optimized.AddBaseline(baseline1)
	optimized.AddBaseline(baseline2)
	optimized.AddBaseline(baseline3)

	// Should have 2 deployment entries but only 1 process set
	assert.Equal(t, 2, len(optimized.deploymentBaselines))
	assert.Equal(t, 1, len(optimized.processSets))

	// Both deployments should reference the same process set
	key1 := optimized.deploymentBaselines["deployment-1"]["container-1"]
	key2 := optimized.deploymentBaselines["deployment-2"]["container-2"]
	assert.Equal(t, key1, key2)

	// Process set should have reference count of 3
	entry := optimized.processSets[key1]
	assert.Equal(t, 3, entry.refCount)

	optimized.RemoveDeployment("deployment-1")

	assert.Equal(t, 1, len(optimized.deploymentBaselines))
	assert.Equal(t, 1, len(optimized.processSets))

	// Process set should have reference count of 1
	// because two have been removed
	entry = optimized.processSets[key1]
	assert.Equal(t, 1, entry.refCount)
}

func TestNilSafety(t *testing.T) {
	// Test that both implementations handle nil ProcessIndicator safely
	testCases := []struct {
		name             string
		evaluatorFactory func() Evaluator
	}{
		{"Original", newBaselineEvaluator},
		{"Optimized", newOptimizedBaselineEvaluator},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			evaluator := tc.evaluatorFactory()

			// Should not panic and should return false (safe default)
			result := evaluator.IsOutsideLockedBaseline(nil)
			assert.False(t, result, "nil ProcessIndicator should be treated as within baseline")
		})
	}
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
