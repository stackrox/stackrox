package node

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	ctx := context.Background()

	mult := NewVulnerabilities()
	nodes := multipliers.GetMockNodes()
	result := mult.Score(ctx, nodes[0])
	assert.Equal(t, float32(1.1815), result.Score)

	// Changing CVSS should not change result.
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(ctx, nodes[0])
	assert.Equal(t, float32(1.1815), result.Score)
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(ctx, nodes[0])
	assert.Equal(t, float32(1.1815), result.Score)

	// Set both severity ratings to unknown and then there should be a nil RiskResult
	nodes[0].GetScan().GetComponents()[0].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	nodes[0].GetScan().GetComponents()[1].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	nodes[0].GetScan().GetComponents()[0].GetVulnerabilities()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	nodes[0].GetScan().GetComponents()[1].GetVulnerabilities()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	result = mult.Score(ctx, nodes[0])
	assert.Nil(t, result)
}
