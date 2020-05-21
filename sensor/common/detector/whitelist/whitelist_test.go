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
	// No whitelist added, nothing is outside a locked whitelist
	assert.False(t, evaluator.IsOutsideLockedWhitelist(process))

	// Add whitelist that does not contain the value
	evaluator.AddWhitelist(notInWhitelist)
	assert.True(t, evaluator.IsOutsideLockedWhitelist(process))

	// Verify that different whitelists produce expected outcomes.
	evaluator.AddWhitelist(inWhitelist)
	assert.False(t, evaluator.IsOutsideLockedWhitelist(process))
	evaluator.AddWhitelist(notInWhitelist)
	assert.True(t, evaluator.IsOutsideLockedWhitelist(process))
	evaluator.AddWhitelist(notInUnlockedWhitelist)
	assert.False(t, evaluator.IsOutsideLockedWhitelist(process))

	// Add locked whitelist then remove deployment
	evaluator.AddWhitelist(notInWhitelist)
	assert.True(t, evaluator.IsOutsideLockedWhitelist(process))
	evaluator.RemoveDeployment(process.GetDeploymentId())
	assert.False(t, evaluator.IsOutsideLockedWhitelist(process))
}
