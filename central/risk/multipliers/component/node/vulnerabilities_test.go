package node

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	ctx := context.Background()

	mult := NewVulnerabilities()
	nodes := multipliers.GetMockNodes()
	result := mult.Score(ctx, nodes[0].GetScan().GetComponents()[0])
	assert.Equal(t, float32(1.08748), result.Score)

	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(ctx, nodes[0].GetScan().GetComponents()[0])
	assert.Nil(t, result)

	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(ctx, nodes[0].GetScan().GetComponents()[0])
	assert.Equal(t, float32(1.3), result.Score)

	// Set both CVSS to 0 and then there should be a nil RiskResult
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	nodes[0].GetScan().GetComponents()[1].GetVulns()[0].Cvss = 0

	for _, nodeComponent := range nodes[0].GetScan().GetComponents() {
		result = mult.Score(ctx, nodeComponent)
		assert.Nil(t, result)
	}
}
