package multipliers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	mult := NewVulnerabilities()
	deployment := getMockDeployment()
	result := mult.Score(deployment)
	// The first deployment to be processed will always be 2.0
	assert.Equal(t, float32(1.05), result.Score)

	// Set the cvss value from 5 -> 0 and rescore should give a a new score of 1.5
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(deployment)
	assert.Equal(t, float32(1.025), result.Score)

	// Set the CVSS to 10 and then new score should be 2.0
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(deployment)
	assert.Equal(t, float32(1.125), result.Score)

	// Set both CVSS to 0 and then there should be a nil RiskResult
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	deployment.GetContainers()[0].GetImage().GetScan().GetComponents()[0].GetVulns()[1].Cvss = 0
	result = mult.Score(deployment)
	assert.Nil(t, result)
}
