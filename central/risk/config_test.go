package risk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreVolumesAndSecrets(t *testing.T) {
	mult := newServiceConfigMultiplier()
	deployment := getMockDeployment()
	volumeFactor, secretFactor := mult.scoreVolumesAndSecrets(deployment)
	assert.NotEmpty(t, volumeFactor)
	assert.NotEmpty(t, secretFactor)

	assert.Contains(t, volumeFactor, "rw volume")
	assert.Contains(t, secretFactor, "secret")

	for _, container := range deployment.GetContainers() {
		container.Volumes = nil
	}
	volumeFactor, secretFactor = mult.scoreVolumesAndSecrets(deployment)
	assert.Empty(t, volumeFactor)
	assert.Empty(t, secretFactor)
}

func TestScoreCapabilities(t *testing.T) {
	mult := newServiceConfigMultiplier()
	deployment := getMockDeployment()
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
	mult := newServiceConfigMultiplier()
	deployment := getMockDeployment()
	factor := mult.scorePrivilege(deployment)
	assert.NotEmpty(t, factor)

	deployment.Containers[0].SecurityContext.Privileged = false
	factor = mult.scorePrivilege(deployment)
	assert.Empty(t, factor)
}

func TestConfigScore(t *testing.T) {
	// Hit all values
	mult := newServiceConfigMultiplier()
	deployment := getMockDeployment()
	result := mult.Score(deployment)
	assert.Equal(t, result.GetScore(), float32(2))
}
