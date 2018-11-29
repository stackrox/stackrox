package multipliers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	mult := NewVulnerabilities()
	deployment := getMockDeployment()
	result := mult.Score(deployment)
	assert.Equal(t, float32(1.3), result.Score)

	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(deployment)
	assert.Equal(t, float32(1.225), result.Score)

	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(deployment)
	assert.Equal(t, float32(1.525), result.Score)

	// Set both CVSS to 0 and then there should be a nil RiskResult
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[1].Cvss = 0
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[1].GetVulns()[0].Cvss = 0
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[1].GetVulns()[1].Cvss = 0
	result = mult.Score(deployment)
	assert.Nil(t, result)
}
