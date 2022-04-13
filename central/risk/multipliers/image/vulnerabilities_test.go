package image

import (
	"context"
	"testing"

	"github.com/stackrox/stackrox/central/risk/multipliers"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	ctx := context.Background()

	mult := NewVulnerabilities()
	images := multipliers.GetMockImages()
	result := mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.Score)

	// Changing CVSS score should not affect result.
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.Score)
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.Score)

	// Set both severity to unknown and then there should be a nil RiskResult
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[0].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	result = mult.Score(ctx, images[0])
	assert.Nil(t, result)
}
