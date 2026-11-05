package image

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/risk/multipliers"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestVulnerabilitiesScore(t *testing.T) {
	ctx := context.Background()

	mult := NewVulnerabilities()
	images := multipliers.GetMockImages()
	result := mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())
	assert.Equal(t, "Image \"docker.io/library/nginx:1.10\" contains 2 distinct CVEs with severities ranging between Low and Critical", result.GetFactors()[0].GetMessage())

	// Changing CVSS score should not affect result.
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())

	// Unknown-severity CVEs are displayed, but do not contribute to the score.
	images[0].GetScan().GetComponents()[0].Vulns = append(images[0].GetScan().GetComponents()[0].GetVulns(),
		&storage.EmbeddedVulnerability{Cve: "CVE-2019-0003", Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY})
	result = mult.Score(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())
	assert.Equal(t, "Image \"docker.io/library/nginx:1.10\" contains 3 distinct CVEs with severities ranging between Low and Critical", result.GetFactors()[0].GetMessage())

	// Only unknown-severity vulnerabilities remain, so there is no risk result.
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[0].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	result = mult.Score(ctx, images[0])
	assert.Nil(t, result)
}

func TestVulnerabilitiesScoreV2(t *testing.T) {
	testutils.MustUpdateFeature(t, features.FlattenImageData, true)
	ctx := context.Background()

	mult := NewVulnerabilities()
	images := multipliers.GetMockImagesV2()
	result := mult.ScoreV2(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())
	assert.Equal(t, "Image \"docker.io/library/nginx:1.10\" contains 2 distinct CVEs with severities ranging between Low and Critical", result.GetFactors()[0].GetMessage())

	// Changing CVSS score should not affect result.
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 0
	result = mult.ScoreV2(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Cvss = 10
	result = mult.ScoreV2(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())

	// Unknown-severity CVEs are displayed, but do not contribute to the score.
	images[0].GetScan().GetComponents()[0].Vulns = append(images[0].GetScan().GetComponents()[0].GetVulns(),
		&storage.EmbeddedVulnerability{Cve: "CVE-2019-0003", Severity: storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY})
	result = mult.ScoreV2(ctx, images[0])
	assert.Equal(t, float32(1.59025), result.GetScore())
	assert.Equal(t, "Image \"docker.io/library/nginx:1.10\" contains 3 distinct CVEs with severities ranging between Low and Critical", result.GetFactors()[0].GetMessage())

	// Only unknown-severity vulnerabilities remain, so there is no risk result.
	images[0].GetScan().GetComponents()[0].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[0].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[0].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	images[0].GetScan().GetComponents()[1].GetVulns()[1].Severity = storage.VulnerabilitySeverity_UNKNOWN_VULNERABILITY_SEVERITY
	result = mult.ScoreV2(ctx, images[0])
	assert.Nil(t, result)
}
