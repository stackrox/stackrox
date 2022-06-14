package deployment

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stretchr/testify/assert"
)

func TestScoreVolumesAndSecrets(t *testing.T) {
	mult := &serviceConfigMultiplier{}
	deployment := multipliers.GetMockDeployment()
	volumeFactor := mult.scoreVolumes(deployment)
	assert.NotEmpty(t, volumeFactor)

	assert.Contains(t, volumeFactor, "rw volume")

	for _, container := range deployment.GetContainers() {
		container.Volumes = nil
	}
	volumeFactor = mult.scoreVolumes(deployment)
	assert.Empty(t, volumeFactor)
}

func TestScoreSecrets(t *testing.T) {
	mult := &serviceConfigMultiplier{}
	deployment := multipliers.GetMockDeployment()
	secretFactor := mult.scoreSecrets(deployment)
	assert.NotEmpty(t, secretFactor)

	assert.Contains(t, secretFactor, "secret")

	for _, container := range deployment.GetContainers() {
		container.Secrets = nil
	}
	secretFactor = mult.scoreSecrets(deployment)
	assert.Empty(t, secretFactor)
}

func TestScoreCapabilities(t *testing.T) {
	mult := &serviceConfigMultiplier{}
	deployment := multipliers.GetMockDeployment()
	addFactor, dropFactor := mult.scoreCapabilities(deployment)
	assert.NotEmpty(t, addFactor)
	assert.NotEmpty(t, dropFactor)

	assert.Contains(t, addFactor, "ALL")
	assert.Contains(t, dropFactor, "No capabilities")

	for _, container := range deployment.GetContainers() {
		container.GetSecurityContext().AddCapabilities = nil
		container.SecurityContext.DropCapabilities = []string{"SYS_MODULE"}
	}
	addFactor, dropFactor = mult.scoreCapabilities(deployment)
	assert.Empty(t, addFactor)
	assert.Empty(t, dropFactor)
}

func TestScorePrivileged(t *testing.T) {
	mult := &serviceConfigMultiplier{}
	deployment := multipliers.GetMockDeployment()
	factor := mult.scorePrivilege(deployment)
	assert.NotEmpty(t, factor)

	deployment.Containers[0].SecurityContext.Privileged = false
	factor = mult.scorePrivilege(deployment)
	assert.Empty(t, factor)
}

func TestConfigScore(t *testing.T) {
	// Hit all values
	mult := &serviceConfigMultiplier{}
	deployment := multipliers.GetMockDeployment()
	result := mult.Score(context.Background(), deployment, nil)
	assert.Equal(t, result.GetScore(), float32(2))
}
