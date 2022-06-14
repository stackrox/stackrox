package baseline

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
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
		UserLockedTimestamp: types.TimestampNow(),
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
		UserLockedTimestamp: types.TimestampNow(),
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
