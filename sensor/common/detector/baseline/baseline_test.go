package baseline

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestBaseline(t *testing.T) {
	process := fixtures.GetProcessIndicator()

	pbk := &storage.ProcessBaselineKey{}
	pbk.SetDeploymentId(process.GetDeploymentId())
	pbk.SetContainerName(process.GetContainerName())
	notInUnlockedBaseline := &storage.ProcessBaseline{}
	notInUnlockedBaseline.SetKey(pbk)

	pbk2 := &storage.ProcessBaselineKey{}
	pbk2.SetDeploymentId(process.GetDeploymentId())
	pbk2.SetContainerName(process.GetContainerName())
	notInBaseline := &storage.ProcessBaseline{}
	notInBaseline.SetKey(pbk2)
	notInBaseline.SetUserLockedTimestamp(protocompat.TimestampNow())

	inBaseline := storage.ProcessBaseline_builder{
		Key: storage.ProcessBaselineKey_builder{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		}.Build(),
		Elements: []*storage.BaselineElement{
			storage.BaselineElement_builder{
				Element: storage.BaselineItem_builder{
					ProcessName: proto.String(process.GetSignal().GetExecFilePath()),
				}.Build(),
			}.Build(),
		},
		UserLockedTimestamp: protocompat.TimestampNow(),
	}.Build()

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
