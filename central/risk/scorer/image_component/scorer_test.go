package imagecomponent

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	imageComponentMultiplier "github.com/stackrox/rox/central/risk/multipliers/image_component"
	pkgScorer "github.com/stackrox/rox/central/risk/scorer"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	ctx := context.Background()

	mockCtrl := gomock.NewController(t)

	imageComponent := pkgScorer.GetMockImage().GetScan().GetComponents()[0]
	scorer := NewImageComponentScorer()

	// Without user defined function
	expectedRiskScore := 1.15
	expectedRiskResults := []*storage.Risk_Result{
		{
			Name: imageComponentMultiplier.ImageComponentVulnerabilitiesHeading,
			Factors: []*storage.Risk_Result_Factor{
				{Message: "Image Component ComponentX version v1 contains 2 CVEs with CVSS scores ranging between 5.0 and 5.0"},
			},
			Score: 1.15,
		},
	}

	actualRisk := scorer.Score(ctx, imageComponent)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	actualRisk = scorer.Score(ctx, imageComponent)
	assert.Equal(t, expectedRiskResults, actualRisk.GetResults())
	assert.InDelta(t, expectedRiskScore, actualRisk.GetScore(), 0.0001)

	mockCtrl.Finish()
}
