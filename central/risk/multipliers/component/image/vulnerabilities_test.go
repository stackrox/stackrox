package image

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	ctx := context.Background()

	mult := NewVulnerabilities()
	images := multipliers.GetMockImages()
	result := mult.Score(ctx, scancomponent.NewFromImageComponent(images[0].GetScan().GetComponents()[0]))
	assert.Equal(t, float32(1.22875), result.Score)

	// Changing CVSS no longer affects score.
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(ctx, scancomponent.NewFromImageComponent(images[0].GetScan().GetComponents()[0]))
	assert.Equal(t, float32(1.22875), result.Score)
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(ctx, scancomponent.NewFromImageComponent(images[0].GetScan().GetComponents()[0]))
	assert.Equal(t, float32(1.22875), result.Score)

	// Set severity to unknown and then there should be a nil RiskResult
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[0].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY

	for _, imgComponent := range images[0].GetScan().GetComponents() {
		result = mult.Score(ctx, scancomponent.NewFromImageComponent(imgComponent))
		assert.Nil(t, result)
	}
}
