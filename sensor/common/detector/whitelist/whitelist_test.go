package whitelist

import (
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stretchr/testify/assert"
)

func TestWhitelist(t *testing.T) {
	process := fixtures.GetProcessIndicator()

	notInUnlockedWhitelist := &storage.ProcessWhitelist{
		Key: &storage.ProcessWhitelistKey{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		},
	}

	notInWhitelist := &storage.ProcessWhitelist{
		Key: &storage.ProcessWhitelistKey{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		},
		UserLockedTimestamp: types.TimestampNow(),
	}

	inWhitelist := &storage.ProcessWhitelist{
		Key: &storage.ProcessWhitelistKey{
			DeploymentId:  process.GetDeploymentId(),
			ContainerName: process.GetContainerName(),
		},
		Elements: []*storage.WhitelistElement{
			{
				Element: &storage.WhitelistItem{
					Item: &storage.WhitelistItem_ProcessName{
						ProcessName: process.GetSignal().GetExecFilePath(),
					},
				},
			},
		},
		UserLockedTimestamp: types.TimestampNow(),
	}

	evaluator := NewWhitelistEvaluator()
	// No whitelist added should return true
	assert.True(t, evaluator.IsInWhitelist(process))

	// Add whitelist that does not container the value
	evaluator.AddWhitelist(notInWhitelist)
	assert.False(t, evaluator.IsInWhitelist(process))

	// Add whitelist that does contain the value
	evaluator.AddWhitelist(inWhitelist)
	assert.True(t, evaluator.IsInWhitelist(process))

	// Re-add the whitelist and then remove the deployment
	evaluator.AddWhitelist(notInWhitelist)
	assert.False(t, evaluator.IsInWhitelist(process))
	evaluator.AddWhitelist(notInUnlockedWhitelist)
	assert.True(t, evaluator.IsInWhitelist(process))

	// Add locked whitelist then remove deployment
	evaluator.AddWhitelist(notInWhitelist)
	assert.False(t, evaluator.IsInWhitelist(process))
	evaluator.RemoveDeployment(process.GetDeploymentId())
	assert.True(t, evaluator.IsInWhitelist(process))
}
