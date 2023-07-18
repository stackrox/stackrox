package image

import (
	"context"
	"testing"

	imageComponentMultiplier "github.com/stackrox/rox/central/risk/multipliers/component/image"
	pkgScorer "github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/scancomponent"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	imageComponent := pkgScorer.GetMockImage().GetScan().GetComponents()[0]
	imageComponent.GetVulns()[0].Severity = storage.VulnerabilitySeverity_LOW_VULNERABILITY_SEVERITY
	imageComponent.GetVulns()[1].ScoreVersion = storage.EmbeddedVulnerability_V3
	imageComponent.GetVulns()[1].Severity = storage.VulnerabilitySeverity_CRITICAL_VULNERABILITY_SEVERITY
	scorer := NewImageComponentScorer()

	// Without user defined function
	expectedRiskScore := 1.5534999
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: imageComponentMultiplier.VulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image Component ComponentX version v1 contains 3 CVEs with severities ranging between Low and Critical"},
			},
			Score: 1.5534999,
		},
	}

	actualRisk := scorer.Score(ctx, scancomponent.NewFromImageComponent(imageComponent), "")
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
