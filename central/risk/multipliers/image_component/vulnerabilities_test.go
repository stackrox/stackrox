package imagecomponent

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	ctx := context.Background()

	mult := NewVulnerabilities()
	images := multipliers.GetMockImages()
	result := mult.Score(ctx, images[0].GetScan().GetComponents()[0])
	assert.Equal(t, float32(1.15), result.Score)

	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(ctx, images[0].GetScan().GetComponents()[0])
	assert.Equal(t, float32(1.075), result.Score)

	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(ctx, images[0].GetScan().GetComponents()[0])
	assert.Equal(t, float32(1.375), result.Score)

	// Set both CVSS to 0 and then there should be a nil RiskResult
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	images[0].GetScan().GetComponents()[0].GetVulns()[1].Cvss = 0
	images[0].GetScan().GetComponents()[1].GetVulns()[0].Cvss = 0
	images[0].GetScan().GetComponents()[1].GetVulns()[1].Cvss = 0

	for _, imgComponent := range images[0].GetScan().GetComponents() {
		result = mult.Score(ctx, imgComponent)
		assert.Nil(t, result)
	}
}
